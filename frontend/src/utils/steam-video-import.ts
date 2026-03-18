const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api'

type MP4BoxFile = {
  onReady?: (info: any) => void
  onError?: (error: any) => void
  onSamples?: (id: number, user: any, samples: any[]) => void
  appendBuffer: (buffer: ArrayBuffer & { fileStart?: number }) => void
  flush: () => void
  setExtractionOptions: (trackId: number, user: any, options: { nbSamples: number }) => void
  start: () => void
  addTrack: (options: Record<string, any>) => number
  addSample: (trackId: number, data: ArrayBuffer | Uint8Array, options: Record<string, any>) => void
  getTrackById: (id: number) => any
  getBuffer: () => ArrayBuffer
}

type MP4BoxStatic = {
  createFile: () => MP4BoxFile
}

type DashRepresentation = {
  id: string
  type: 'video' | 'audio'
  width?: number
  height?: number
  bandwidth: number
  codec?: string
  init: string
  mediaTemplate: string
  segmentDuration: number
}

let mp4BoxLoader: Promise<MP4BoxStatic> | null = null

function getMP4BoxGlobal(): MP4BoxStatic | null {
  return (window as any).MP4Box || null
}

async function ensureMP4Box(): Promise<MP4BoxStatic> {
  const existing = getMP4BoxGlobal()
  if (existing) return existing
  if (mp4BoxLoader) return mp4BoxLoader

  mp4BoxLoader = new Promise((resolve, reject) => {
    const script = document.createElement('script')
    script.src = '/vendor/mp4box.min.js'
    script.async = true
    script.onload = () => {
      const loaded = getMP4BoxGlobal()
      if (loaded) {
        resolve(loaded)
        return
      }
      reject(new Error('MP4Box loaded but global is unavailable'))
    }
    script.onerror = () => reject(new Error('Failed to load MP4Box'))
    document.head.appendChild(script)
  })

  return mp4BoxLoader
}

function parseIsoDuration(value: string | null): number {
  if (!value) return 60
  const hours = value.match(/(\d+\.?\d*)H/)
  const minutes = value.match(/(\d+\.?\d*)M/)
  const seconds = value.match(/(\d+\.?\d*)S/)
  const total =
    (hours ? parseFloat(hours[1]) * 3600 : 0) +
    (minutes ? parseFloat(minutes[1]) * 60 : 0) +
    (seconds ? parseFloat(seconds[1]) : 0)
  return total > 0 ? total : 60
}

async function proxyFetchArrayBuffer(url: string): Promise<ArrayBuffer> {
  const response = await fetch(`${API_BASE_URL}/steam/proxy?url=${encodeURIComponent(url)}`, {
    credentials: 'same-origin',
  })
  if (!response.ok) {
    throw new Error(`Steam proxy failed: ${response.status}`)
  }
  return await response.arrayBuffer()
}

async function proxyFetchText(url: string): Promise<string> {
  const response = await fetch(`${API_BASE_URL}/steam/proxy?url=${encodeURIComponent(url)}`, {
    credentials: 'same-origin',
  })
  if (!response.ok) {
    throw new Error(`Steam proxy failed: ${response.status}`)
  }
  return await response.text()
}

async function parseMPD(mpdUrl: string): Promise<{ representations: DashRepresentation[]; totalDuration: number }> {
  const text = await proxyFetchText(mpdUrl)
  const parser = new DOMParser()
  const xml = parser.parseFromString(text, 'application/xml')
  const baseUrl = mpdUrl.substring(0, mpdUrl.lastIndexOf('/') + 1)
  const representations: DashRepresentation[] = []

  const appendRepresentations = (selector: string, type: 'video' | 'audio') => {
    xml.querySelectorAll(selector).forEach((rep) => {
      const parentTemplate = rep.parentElement?.querySelector(':scope > SegmentTemplate')
      const ownTemplate = rep.querySelector(':scope > SegmentTemplate')
      const segTemplate = ownTemplate || parentTemplate
      if (!segTemplate) return

      const id = rep.getAttribute('id')
      const initTemplate = segTemplate.getAttribute('initialization')
      const mediaTemplate = segTemplate.getAttribute('media')
      if (!id || !initTemplate || !mediaTemplate) return

      const timescale = parseInt(segTemplate.getAttribute('timescale') || '1000000', 10)
      const duration = parseInt(segTemplate.getAttribute('duration') || '0', 10)
      if (!duration || !timescale) return

      representations.push({
        id,
        type,
        width: rep.getAttribute('width') ? parseInt(rep.getAttribute('width') || '0', 10) : undefined,
        height: rep.getAttribute('height') ? parseInt(rep.getAttribute('height') || '0', 10) : undefined,
        bandwidth: parseInt(rep.getAttribute('bandwidth') || '0', 10),
        codec: rep.getAttribute('codecs') || undefined,
        init: baseUrl + initTemplate.replace('$RepresentationID$', id),
        mediaTemplate: baseUrl + mediaTemplate,
        segmentDuration: duration / timescale,
      })
    })
  }

  appendRepresentations('AdaptationSet[contentType="video"] Representation, AdaptationSet[mimeType^="video"] Representation', 'video')
  appendRepresentations('AdaptationSet[contentType="audio"] Representation, AdaptationSet[mimeType^="audio"] Representation', 'audio')

  const totalDuration = parseIsoDuration(xml.querySelector('MPD')?.getAttribute('mediaPresentationDuration') || null)
  return { representations, totalDuration }
}

async function downloadSegments(
  representation: DashRepresentation,
  totalDuration: number,
  onProgress?: (percent: number) => void,
): Promise<ArrayBuffer[]> {
  const chunks: ArrayBuffer[] = []
  chunks.push(await proxyFetchArrayBuffer(representation.init))

  const numberFormat = representation.mediaTemplate.match(/\$Number%0(\d+)d\$/)
  const numSegments = Math.ceil(totalDuration / representation.segmentDuration)

  for (let index = 1; index <= numSegments; index++) {
    let segmentUrl = representation.mediaTemplate.replace('$RepresentationID$', representation.id)
    if (numberFormat) {
      segmentUrl = segmentUrl.replace(numberFormat[0], String(index).padStart(parseInt(numberFormat[1], 10), '0'))
    } else {
      segmentUrl = segmentUrl.replace('$Number$', String(index))
    }

    const chunk = await proxyFetchArrayBuffer(segmentUrl)
    chunks.push(chunk)
    onProgress?.((index / numSegments) * 100)
  }

  return chunks
}

function concatenateBuffers(buffers: ArrayBuffer[]): ArrayBuffer {
  const totalLength = buffers.reduce((sum, item) => sum + item.byteLength, 0)
  const output = new Uint8Array(totalLength)
  let offset = 0
  for (const buffer of buffers) {
    output.set(new Uint8Array(buffer), offset)
    offset += buffer.byteLength
  }
  return output.buffer
}

function findSubBox(data: Uint8Array, targetType: string): Uint8Array | null {
  const view = new DataView(data.buffer, data.byteOffset, data.byteLength)
  let offset = 8
  while (offset < data.length - 8) {
    const size = view.getUint32(offset)
    const type = String.fromCharCode(data[offset + 4], data[offset + 5], data[offset + 6], data[offset + 7])
    if (size < 8 || offset + size > data.length) break
    if (type === targetType) {
      return data.slice(offset, offset + size)
    }
    offset += size
  }
  return null
}

function findSubBoxWithOffset(data: Uint8Array, targetType: string): { offset: number; size: number; data: Uint8Array } | null {
  const view = new DataView(data.buffer, data.byteOffset, data.byteLength)
  let offset = 8
  while (offset < data.length - 8) {
    const size = view.getUint32(offset)
    const type = String.fromCharCode(data[offset + 4], data[offset + 5], data[offset + 6], data[offset + 7])
    if (size < 8 || offset + size > data.length) break
    if (type === targetType) {
      return { offset, size, data: data.slice(offset, offset + size) }
    }
    offset += size
  }
  return null
}

function updateTrackIdInTrak(trakData: Uint8Array, newTrackId: number) {
  const view = new DataView(trakData.buffer, trakData.byteOffset, trakData.byteLength)
  let offset = 8
  while (offset < trakData.length - 8) {
    const size = view.getUint32(offset)
    const type = String.fromCharCode(trakData[offset + 4], trakData[offset + 5], trakData[offset + 6], trakData[offset + 7])
    if (size < 8 || offset + size > trakData.length) break
    if (type === 'tkhd') {
      const version = trakData[offset + 8]
      const trackIdOffset = offset + (version === 0 ? 20 : 28)
      view.setUint32(trackIdOffset, newTrackId)
      return
    }
    offset += size
  }
}

function updateTrackIdInTrex(trexData: Uint8Array, newTrackId: number) {
  if (trexData.length < 16) return
  const view = new DataView(trexData.buffer, trexData.byteOffset, trexData.byteLength)
  view.setUint32(12, newTrackId)
}

function updateTrackIdInSidx(sidxData: Uint8Array, newTrackId: number) {
  if (sidxData.length < 16) return
  const view = new DataView(sidxData.buffer, sidxData.byteOffset, sidxData.byteLength)
  view.setUint32(12, newTrackId)
}

function updateTrackIdInMoof(moofData: Uint8Array, newTrackId: number) {
  const view = new DataView(moofData.buffer, moofData.byteOffset, moofData.byteLength)
  let offset = 8
  while (offset < moofData.length - 8) {
    const size = view.getUint32(offset)
    const type = String.fromCharCode(moofData[offset + 4], moofData[offset + 5], moofData[offset + 6], moofData[offset + 7])
    if (size < 8 || offset + size > moofData.length) break
    if (type === 'traf') {
      let trafOffset = offset + 8
      while (trafOffset < offset + size - 8) {
        const subSize = view.getUint32(trafOffset)
        const subType = String.fromCharCode(
          moofData[trafOffset + 4],
          moofData[trafOffset + 5],
          moofData[trafOffset + 6],
          moofData[trafOffset + 7],
        )
        if (subSize < 8) break
        if (subType === 'tfhd') {
          view.setUint32(trafOffset + 12, newTrackId)
          return
        }
        trafOffset += subSize
      }
    }
    offset += size
  }
}

function createMergedMoov(videoMoovData: Uint8Array, audioTrakBox: Uint8Array, audioTrexBox: Uint8Array | null): Uint8Array {
  const audioTrakCopy = new Uint8Array(audioTrakBox)
  updateTrackIdInTrak(audioTrakCopy, 2)

  const videoMvex = findSubBoxWithOffset(videoMoovData, 'mvex')
  if (!videoMvex || !audioTrexBox) {
    const newSize = videoMoovData.length + audioTrakCopy.length
    const result = new Uint8Array(newSize)
    result.set(videoMoovData)
    result.set(audioTrakCopy, videoMoovData.length)
    new DataView(result.buffer).setUint32(0, newSize)
    return result
  }

  const audioTrexCopy = new Uint8Array(audioTrexBox)
  updateTrackIdInTrex(audioTrexCopy, 2)

  const beforeMvex = videoMoovData.slice(0, videoMvex.offset)
  const oldMvex = videoMoovData.slice(videoMvex.offset, videoMvex.offset + videoMvex.size)

  const newMvexSize = videoMvex.size + audioTrexCopy.length
  const newMvex = new Uint8Array(newMvexSize)
  newMvex.set(oldMvex)
  newMvex.set(audioTrexCopy, videoMvex.size)
  new DataView(newMvex.buffer).setUint32(0, newMvexSize)

  const newMoovSize = beforeMvex.length + audioTrakCopy.length + newMvex.length
  const result = new Uint8Array(newMoovSize)
  let offset = 0
  result.set(beforeMvex, offset)
  offset += beforeMvex.length
  result.set(audioTrakCopy, offset)
  offset += audioTrakCopy.length
  result.set(newMvex, offset)
  new DataView(result.buffer).setUint32(0, newMoovSize)
  return result
}

async function binaryMuxFragmentedMP4(
  videoData: ArrayBuffer,
  audioData: ArrayBuffer,
  onStatus?: (status: string) => void,
): Promise<ArrayBuffer> {
  onStatus?.('分析 MP4 结构')
  const parseBoxes = (view: DataView) => {
    const boxes: Array<{ size: number; type: string; offset: number }> = []
    let offset = 0
    while (offset < view.byteLength) {
      if (offset + 8 > view.byteLength) break
      let size = view.getUint32(offset)
      const type = String.fromCharCode(
        view.getUint8(offset + 4),
        view.getUint8(offset + 5),
        view.getUint8(offset + 6),
        view.getUint8(offset + 7),
      )
      if (size === 0) size = view.byteLength - offset
      if (size < 8) break
      boxes.push({ size, type, offset })
      offset += size
    }
    return boxes
  }

  const videoBoxes = parseBoxes(new DataView(videoData))
  const audioBoxes = parseBoxes(new DataView(audioData))
  const videoFtyp = videoBoxes.find((box) => box.type === 'ftyp')
  const videoMoov = videoBoxes.find((box) => box.type === 'moov')
  const audioMoov = audioBoxes.find((box) => box.type === 'moov')
  if (!videoFtyp || !videoMoov || !audioMoov) {
    throw new Error('缺少必要的 MP4 box')
  }

  const videoMoovData = new Uint8Array(videoData, videoMoov.offset, videoMoov.size)
  const audioMoovData = new Uint8Array(audioData, audioMoov.offset, audioMoov.size)
  const audioTrakBox = findSubBox(audioMoovData, 'trak')
  const audioMvexBox = findSubBox(audioMoovData, 'mvex')
  const audioTrexBox = audioMvexBox ? findSubBox(audioMvexBox, 'trex') : null
  if (!audioTrakBox) {
    throw new Error('未找到音频轨')
  }

  const mergedMoov = createMergedMoov(videoMoovData, audioTrakBox, audioTrexBox)
  const videoFragments: Uint8Array[] = []
  for (const box of videoBoxes) {
    if (['styp', 'sidx', 'moof', 'mdat'].includes(box.type)) {
      videoFragments.push(new Uint8Array(videoData, box.offset, box.size))
    }
  }

  const audioFragments: Uint8Array[] = []
  for (const box of audioBoxes) {
    if (box.type === 'styp' || box.type === 'sidx') {
      const fragment = new Uint8Array(audioData, box.offset, box.size)
      if (box.type === 'sidx') updateTrackIdInSidx(fragment, 2)
      audioFragments.push(fragment)
    } else if (box.type === 'moof') {
      const fragment = new Uint8Array(audioData.slice(box.offset, box.offset + box.size))
      updateTrackIdInMoof(fragment, 2)
      audioFragments.push(fragment)
    } else if (box.type === 'mdat') {
      audioFragments.push(new Uint8Array(audioData, box.offset, box.size))
    }
  }

  const ftypData = new Uint8Array(videoData, videoFtyp.offset, videoFtyp.size)
  let totalSize = ftypData.length + mergedMoov.length
  for (const fragment of videoFragments) totalSize += fragment.length
  for (const fragment of audioFragments) totalSize += fragment.length

  const output = new Uint8Array(totalSize)
  let writeOffset = 0
  output.set(ftypData, writeOffset)
  writeOffset += ftypData.length
  output.set(mergedMoov, writeOffset)
  writeOffset += mergedMoov.length
  for (const fragment of videoFragments) {
    output.set(fragment, writeOffset)
    writeOffset += fragment.length
  }
  for (const fragment of audioFragments) {
    output.set(fragment, writeOffset)
    writeOffset += fragment.length
  }
  return output.buffer
}

function extractStsdFromMoov(data: ArrayBuffer): Uint8Array | null {
  const view = new DataView(data)
  let offset = 0
  while (offset < data.byteLength - 8) {
    const size = view.getUint32(offset)
    const type = String.fromCharCode(
      view.getUint8(offset + 4),
      view.getUint8(offset + 5),
      view.getUint8(offset + 6),
      view.getUint8(offset + 7),
    )
    if (size < 8 || offset + size > data.byteLength) break
    if (type === 'moov') {
      return findStsdInBox(new Uint8Array(data, offset, size), 8)
    }
    offset += size
  }
  return null
}

function findStsdInBox(data: Uint8Array, startOffset: number): Uint8Array | null {
  const view = new DataView(data.buffer, data.byteOffset, data.byteLength)
  let offset = startOffset
  while (offset < data.length - 8) {
    const size = view.getUint32(offset)
    const type = String.fromCharCode(data[offset + 4], data[offset + 5], data[offset + 6], data[offset + 7])
    if (size < 8 || offset + size > data.length) break
    if (type === 'stsd') {
      return data.slice(offset, offset + size)
    }
    if (['trak', 'mdia', 'minf', 'stbl'].includes(type)) {
      const result = findStsdInBox(data.slice(offset, offset + size), 8)
      if (result) return result
    }
    offset += size
  }
  return null
}

function buildMP4WithRawStsd(tracks: any[]): ArrayBuffer {
  const makeBox = (type: string, content: Uint8Array) => {
    const size = 8 + content.length
    const box = new Uint8Array(size)
    const view = new DataView(box.buffer)
    view.setUint32(0, size)
    box[4] = type.charCodeAt(0)
    box[5] = type.charCodeAt(1)
    box[6] = type.charCodeAt(2)
    box[7] = type.charCodeAt(3)
    box.set(content, 8)
    return box
  }
  const concat = (...arrays: Uint8Array[]) => {
    const total = arrays.reduce((sum, item) => sum + item.length, 0)
    const result = new Uint8Array(total)
    let offset = 0
    for (const array of arrays) {
      result.set(array, offset)
      offset += array.length
    }
    return result
  }

  const ftypContent = new Uint8Array(20)
  ftypContent.set([0x69, 0x73, 0x6f, 0x6d], 0)
  new DataView(ftypContent.buffer).setUint32(4, 0x200)
  ftypContent.set([0x69, 0x73, 0x6f, 0x6d, 0x69, 0x73, 0x6f, 0x32, 0x61, 0x76, 0x63, 0x31], 8)
  const ftyp = makeBox('ftyp', ftypContent)

  const movieTimescale = 1000
  let maxDuration = 0
  for (const track of tracks) {
    const duration = (track.duration * movieTimescale) / track.timescale
    if (duration > maxDuration) maxDuration = duration
  }

  const mvhdContent = new Uint8Array(100)
  const mvhdView = new DataView(mvhdContent.buffer)
  mvhdView.setUint32(12, movieTimescale)
  mvhdView.setUint32(16, Math.round(maxDuration))
  mvhdView.setUint32(20, 0x00010000)
  mvhdView.setUint16(24, 0x0100)
  mvhdView.setUint32(36, 0x00010000)
  mvhdView.setUint32(52, 0x00010000)
  mvhdView.setUint32(68, 0x40000000)
  mvhdView.setUint32(96, tracks.length + 1)
  const mvhd = makeBox('mvhd', mvhdContent)

  const allSampleData: Uint8Array[] = []
  for (const track of tracks) {
    for (const sample of track.samples) {
      if (sample.data) allSampleData.push(new Uint8Array(sample.data))
    }
  }
  const mdatContent = concat(...allSampleData)

  const buildTrak = (track: any, dataOffset: number) => {
    const trackDuration = Math.round((track.duration * movieTimescale) / track.timescale)
    const tkhdContent = new Uint8Array(84)
    const tkhdView = new DataView(tkhdContent.buffer)
    tkhdView.setUint32(0, 0x00000003)
    tkhdView.setUint32(12, track.id)
    tkhdView.setUint32(20, trackDuration)
    tkhdView.setUint16(36, track.type === 'audio' ? 0x0100 : 0)
    tkhdView.setUint32(40, 0x00010000)
    tkhdView.setUint32(56, 0x00010000)
    tkhdView.setUint32(72, 0x40000000)
    tkhdView.setUint32(76, track.width << 16)
    tkhdView.setUint32(80, track.height << 16)
    const tkhd = makeBox('tkhd', tkhdContent)

    const mdhdContent = new Uint8Array(24)
    const mdhdView = new DataView(mdhdContent.buffer)
    mdhdView.setUint32(12, track.timescale)
    mdhdView.setUint32(16, track.duration)
    mdhdView.setUint16(20, 0x55c4)
    const mdhd = makeBox('mdhd', mdhdContent)

    const handlerType = track.type === 'video' ? 'vide' : 'soun'
    const handlerName = track.type === 'video' ? 'VideoHandler' : 'SoundHandler'
    const hdlrContent = new Uint8Array(25 + handlerName.length)
    hdlrContent[8] = handlerType.charCodeAt(0)
    hdlrContent[9] = handlerType.charCodeAt(1)
    hdlrContent[10] = handlerType.charCodeAt(2)
    hdlrContent[11] = handlerType.charCodeAt(3)
    for (let i = 0; i < handlerName.length; i++) hdlrContent[24 + i] = handlerName.charCodeAt(i)
    const hdlr = makeBox('hdlr', hdlrContent)

    const mediaHeader = track.type === 'video'
      ? makeBox('vmhd', new Uint8Array([0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0]))
      : makeBox('smhd', new Uint8Array(8))
    const urlBox = makeBox('url ', new Uint8Array([0, 0, 0, 1]))
    const dref = makeBox('dref', concat(new Uint8Array([0, 0, 0, 0, 0, 0, 0, 1]), urlBox))
    const dinf = makeBox('dinf', dref)
    const stsd = track.rawStsd || makeBox('stsd', new Uint8Array(8))

    const sttsEntries: Array<{ count: number; duration: number }> = []
    let currentDuration = track.samples[0]?.duration || 1
    let count = 0
    for (const sample of track.samples) {
      if (sample.duration === currentDuration) count++
      else {
        sttsEntries.push({ count, duration: currentDuration })
        currentDuration = sample.duration
        count = 1
      }
    }
    if (count > 0) sttsEntries.push({ count, duration: currentDuration })
    const sttsContent = new Uint8Array(8 + sttsEntries.length * 8)
    const sttsView = new DataView(sttsContent.buffer)
    sttsView.setUint32(4, sttsEntries.length)
    for (let i = 0; i < sttsEntries.length; i++) {
      sttsView.setUint32(8 + i * 8, sttsEntries[i].count)
      sttsView.setUint32(12 + i * 8, sttsEntries[i].duration)
    }
    const stts = makeBox('stts', sttsContent)

    const stscContent = new Uint8Array(20)
    const stscView = new DataView(stscContent.buffer)
    stscView.setUint32(4, 1)
    stscView.setUint32(8, 1)
    stscView.setUint32(12, track.samples.length)
    stscView.setUint32(16, 1)
    const stsc = makeBox('stsc', stscContent)

    const stszContent = new Uint8Array(12 + track.samples.length * 4)
    const stszView = new DataView(stszContent.buffer)
    stszView.setUint32(8, track.samples.length)
    for (let i = 0; i < track.samples.length; i++) {
      stszView.setUint32(12 + i * 4, track.samples[i].data?.byteLength || 0)
    }
    const stsz = makeBox('stsz', stszContent)

    const stcoContent = new Uint8Array(12)
    const stcoView = new DataView(stcoContent.buffer)
    stcoView.setUint32(4, 1)
    stcoView.setUint32(8, dataOffset)
    const stco = makeBox('stco', stcoContent)

    const stblParts: Uint8Array[] = [stsd, stts, stsc, stsz, stco]
    const syncSamples: number[] = []
    if (track.type === 'video') {
      for (let i = 0; i < track.samples.length; i++) {
        if (track.samples[i].is_sync) syncSamples.push(i + 1)
      }
      if (syncSamples.length > 0 && syncSamples.length < track.samples.length) {
        const stssContent = new Uint8Array(8 + syncSamples.length * 4)
        const stssView = new DataView(stssContent.buffer)
        stssView.setUint32(4, syncSamples.length)
        for (let i = 0; i < syncSamples.length; i++) stssView.setUint32(8 + i * 4, syncSamples[i])
        stblParts.push(makeBox('stss', stssContent))
      }
    }

    const needsCtts = track.samples.some((sample: any) => sample.cts !== sample.dts)
    if (needsCtts) {
      const cttsEntries: Array<{ count: number; offset: number }> = []
      let currentOffset = (track.samples[0]?.cts || 0) - (track.samples[0]?.dts || 0)
      let cttsCount = 0
      for (const sample of track.samples) {
        const offset = (sample.cts || 0) - (sample.dts || 0)
        if (offset === currentOffset) cttsCount++
        else {
          cttsEntries.push({ count: cttsCount, offset: currentOffset })
          currentOffset = offset
          cttsCount = 1
        }
      }
      if (cttsCount > 0) cttsEntries.push({ count: cttsCount, offset: currentOffset })
      const cttsContent = new Uint8Array(8 + cttsEntries.length * 8)
      const cttsView = new DataView(cttsContent.buffer)
      cttsView.setUint32(4, cttsEntries.length)
      for (let i = 0; i < cttsEntries.length; i++) {
        cttsView.setUint32(8 + i * 8, cttsEntries[i].count)
        cttsView.setInt32(12 + i * 8, cttsEntries[i].offset)
      }
      stblParts.push(makeBox('ctts', cttsContent))
    }

    const stbl = makeBox('stbl', concat(...stblParts))
    const minf = makeBox('minf', concat(mediaHeader, dinf, stbl))
    const mdia = makeBox('mdia', concat(mdhd, hdlr, minf))
    return makeBox('trak', concat(tkhd, mdia))
  }

  const tempTraks = tracks.map((track) => buildTrak(track, 0))
  const tempMoov = makeBox('moov', concat(mvhd, ...tempTraks))
  let offset = ftyp.length + tempMoov.length + 8
  const finalTraks: Uint8Array[] = []
  for (const track of tracks) {
    finalTraks.push(buildTrak(track, offset))
    for (const sample of track.samples) {
      if (sample.data) offset += sample.data.byteLength
    }
  }

  const moov = makeBox('moov', concat(mvhd, ...finalTraks))
  const mdat = makeBox('mdat', mdatContent)
  return concat(ftyp, moov, mdat).buffer
}

async function defragmentWithRawStsd(
  mp4box: MP4BoxStatic,
  videoData: ArrayBuffer,
  audioData: ArrayBuffer,
  onStatus?: (status: string) => void,
): Promise<ArrayBuffer> {
  return await new Promise((resolve, reject) => {
    const videoStsd = extractStsdFromMoov(videoData)
    const audioStsd = extractStsdFromMoov(audioData)
    if (!videoStsd) {
      reject(new Error('无法提取视频 stsd'))
      return
    }

    const file = mp4box.createFile()
    let fileInfo: any = null
    const trackSamples: Record<number, any[]> = {}
    let resolved = false

    file.onReady = (info) => {
      fileInfo = info
      for (const track of info.tracks) {
        trackSamples[track.id] = []
        file.setExtractionOptions(track.id, null, { nbSamples: track.nb_samples })
      }
      file.start()
    }

    file.onSamples = (trackId, _user, samples) => {
      if (resolved) return
      trackSamples[trackId].push(...samples)
      const allDone = fileInfo && fileInfo.tracks.every((track: any) => (trackSamples[track.id] || []).length >= track.nb_samples)
      if (allDone) {
        resolved = true
        try {
          onStatus?.('构建标准 MP4')
          const tracks = fileInfo.tracks
            .map((track: any, index: number) => {
              const samples = (trackSamples[track.id] || []).filter((sample) => sample.data && sample.data.byteLength > 0)
              if (samples.length === 0) return null
              const duration = track.duration || samples.reduce((sum: number, sample: any) => sum + (sample.duration || 0), 0)
              return {
                id: index + 1,
                type: track.type,
                timescale: track.timescale,
                duration,
                width: track.video?.width || track.track_width || 0,
                height: track.video?.height || track.track_height || 0,
                samples,
                rawStsd: track.type === 'video' ? videoStsd : audioStsd,
              }
            })
            .filter(Boolean)
            .sort((left: any, right: any) => {
              if (left.type === right.type) return 0
              return left.type === 'video' ? -1 : 1
            })
          resolve(buildMP4WithRawStsd(tracks))
        } catch (error) {
          reject(error)
        }
      }
    }

    file.onError = (error) => {
      if (!resolved) {
        resolved = true
        reject(error)
      }
    }

    binaryMuxFragmentedMP4(videoData, audioData, onStatus)
      .then((fragmented) => {
        const buffer = fragmented.slice(0) as ArrayBuffer & { fileStart?: number }
        buffer.fileStart = 0
        file.appendBuffer(buffer)
        file.flush()
      })
      .catch(reject)

    window.setTimeout(() => {
      if (!resolved) {
        resolved = true
        reject(new Error('合并超时'))
      }
    }, 20000)
  })
}

async function parseFile(mp4box: MP4BoxStatic, data: ArrayBuffer): Promise<{ info: any; samples: any[]; file: MP4BoxFile | null }> {
  return await new Promise((resolve, reject) => {
    const file = mp4box.createFile()
    let info: any = null
    const samples: any[] = []
    let extractionDone = false
    let totalSamples = 0

    file.onReady = (fileInfo) => {
      info = fileInfo
      if (fileInfo?.tracks?.length > 0) {
        const track = fileInfo.tracks[0]
        totalSamples = track.nb_samples || 0
        file.setExtractionOptions(track.id, null, { nbSamples: totalSamples })
        file.start()
      } else {
        extractionDone = true
        resolve({ info: null, samples: [], file: null })
      }
    }

    file.onSamples = (_id, _user, batch) => {
      samples.push(...batch)
      if (samples.length >= totalSamples && !extractionDone) {
        extractionDone = true
        resolve({ info, samples, file })
      }
    }

    file.onError = (error) => reject(error)

    const buffer = data.slice(0) as ArrayBuffer & { fileStart?: number }
    buffer.fileStart = 0
    file.appendBuffer(buffer)
    file.flush()

    window.setTimeout(() => {
      if (!extractionDone) {
        extractionDone = true
        resolve({ info, samples, file })
      }
    }, 3000)
  })
}

async function muxVideoAudio(
  videoData: ArrayBuffer,
  audioData: ArrayBuffer,
  onStatus?: (status: string) => void,
): Promise<ArrayBuffer> {
  const mp4box = await ensureMP4Box()
  onStatus?.('解析流信息')

  try {
    onStatus?.('转换为标准 MP4')
    const result = await defragmentWithRawStsd(mp4box, videoData, audioData, onStatus)
    if (result && result.byteLength > videoData.byteLength * 0.9) {
      return result
    }
  } catch {
  }

  try {
    onStatus?.('尝试 fragmented mux')
    const fragmented = await binaryMuxFragmentedMP4(videoData, audioData, onStatus)
    if (fragmented && fragmented.byteLength > videoData.byteLength * 0.9) {
      return fragmented
    }
  } catch {
  }

  onStatus?.('解析视频轨')
  const videoResult = await parseFile(mp4box, videoData)
  onStatus?.('解析音频轨')
  const audioResult = await parseFile(mp4box, audioData)
  if (!videoResult.info?.tracks?.length) throw new Error('无法解析视频轨')
  if (!audioResult.info?.tracks?.length || audioResult.samples.length === 0) throw new Error('无法解析音频轨')

  const outputFile = mp4box.createFile()
  const videoTrack = videoResult.info.tracks[0]
  const audioTrack = audioResult.info.tracks[0]
  let videoDescription: any = null
  let avcC: any = null
  if (videoResult.file) {
    const trak = videoResult.file.getTrackById(videoTrack.id)
    const entry = trak?.mdia?.minf?.stbl?.stsd?.entries?.[0]
    if (entry) {
      videoDescription = entry
      avcC = entry.avcC
    }
  }
  const outputVideoTrackId = outputFile.addTrack({
    type: videoTrack.type || 'video',
    timescale: videoTrack.timescale || 90000,
    duration: videoTrack.duration || 0,
    width: videoTrack.video?.width || videoTrack.track_width || 1920,
    height: videoTrack.video?.height || videoTrack.track_height || 1080,
    brands: ['isom', 'iso2', 'avc1', 'mp41'],
    avcDecoderConfigRecord: avcC,
    description: videoDescription,
  })
  for (const sample of videoResult.samples) {
    outputFile.addSample(outputVideoTrackId, sample.data, {
      duration: sample.duration,
      dts: sample.dts,
      cts: sample.cts,
      is_sync: sample.is_sync,
    })
  }

  let audioDescription: any = null
  if (audioResult.file) {
    const trak = audioResult.file.getTrackById(audioTrack.id)
    audioDescription = trak?.mdia?.minf?.stbl?.stsd?.entries?.[0] || null
  }
  const outputAudioTrackId = outputFile.addTrack({
    type: 'audio',
    timescale: audioTrack.timescale || audioTrack.audio?.sample_rate || 48000,
    duration: audioTrack.duration || 0,
    media_duration: audioTrack.movie_duration || audioTrack.duration || 0,
    samplerate: audioTrack.audio?.sample_rate || 48000,
    channel_count: audioTrack.audio?.channel_count || 2,
    samplesize: 16,
    hdlr: 'soun',
    name: 'SoundHandler',
    description: audioDescription,
  })
  for (const sample of audioResult.samples) {
    outputFile.addSample(outputAudioTrackId, sample.data, {
      duration: sample.duration,
      dts: sample.dts,
      cts: sample.cts,
      is_sync: sample.is_sync,
    })
  }

  const output = outputFile.getBuffer()
  if (!output || output.byteLength < videoData.byteLength * 0.9) {
    throw new Error('预告片合并失败')
  }
  return output
}

function sanitizeName(value: string): string {
  return value.replace(/[^a-zA-Z0-9]+/g, '_').replace(/^_+|_+$/g, '') || 'Steam'
}

async function fetchDirectVideo(url: string): Promise<Blob> {
  const buffer = await proxyFetchArrayBuffer(url)
  const lowered = url.toLowerCase()
  const contentType = lowered.includes('.webm') ? 'video/webm' : 'video/mp4'
  return new Blob([buffer], { type: contentType })
}

export async function importSteamVideoAsFile(options: {
  url: string
  gameName: string
  label: string
  onProgress?: (percent: number, status: string) => void
}): Promise<File> {
  const { url, gameName, label, onProgress } = options
  const safeGameName = sanitizeName(gameName)
  const safeLabel = sanitizeName(label)
  const lowered = url.toLowerCase()

  if (!lowered.includes('.mpd') && !lowered.includes('.m3u8')) {
    onProgress?.(20, '正在下载预告片')
    const blob = await fetchDirectVideo(url)
    onProgress?.(100, '预告片已下载')
    const ext = blob.type.includes('webm') ? 'webm' : 'mp4'
    return new File([blob], `${safeGameName}_${safeLabel}.${ext}`, { type: blob.type })
  }

  onProgress?.(2, '正在解析 DASH 清单')
  const { representations, totalDuration } = await parseMPD(url)
  const videos = representations.filter((item) => item.type === 'video').sort((a, b) => (b.height || 0) - (a.height || 0))
  const audios = representations.filter((item) => item.type === 'audio').sort((a, b) => b.bandwidth - a.bandwidth)

  if (videos.length === 0) {
    throw new Error('DASH 未返回视频轨')
  }

  const bestVideo = videos[0]
  const bestAudio = audios[0]

  onProgress?.(4, '正在下载视频轨')
  const videoChunks = await downloadSegments(bestVideo, totalDuration, (percent) => {
    onProgress?.(4 + percent * 0.4, '正在下载视频轨')
  })
  const videoData = concatenateBuffers(videoChunks)

  if (!bestAudio) {
    throw new Error('DASH 未返回音频轨')
  }

  onProgress?.(46, '正在下载音频轨')
  const audioChunks = await downloadSegments(bestAudio, totalDuration, (percent) => {
    onProgress?.(46 + percent * 0.34, '正在下载音频轨')
  })
  const audioData = concatenateBuffers(audioChunks)

  const muxed = await muxVideoAudio(videoData, audioData, (status) => {
    onProgress?.(85, status)
  })
  onProgress?.(100, '预告片处理完成')

  return new File([muxed], `${safeGameName}_${safeLabel}.mp4`, { type: 'video/mp4' })
}
