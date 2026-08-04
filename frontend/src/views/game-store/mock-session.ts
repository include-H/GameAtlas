/**
 * 游戏店第一阶段 mock 数据。
 *
 * 说明：当前直接使用 NAS 真实游戏库（192.168.1.4:3000）的封面地址，
 * 让第一版静态场景有真实游戏盒效果。后续接入 API 后，这个文件会被
 * “随机 40 个游戏”的真实 Session 取代。
 */
export interface GameStoreMockGame {
  publicId: string
  title: string
  year: string
  platform: string
  coverUrl: string
}

const NAS_API_BASE = 'http://192.168.1.4:3000'

/**
 * CRT 电视当前播放的视频（mock）。
 * 后续接入 API 后，会从本次 Session 的 40 个游戏里随机选一个游戏，
 * 再取它的 preview_videos 第一段播放；没有视频时回退到截图/封面轮播。
 */
export const gameStoreCrtMock = {
  title: '泰坦陨落2',
  videoUrl: `${NAS_API_BASE}/assets/08ae39f9-0630-4bbf-bd46-fb0d2bed4985/358a6d03-054b-437c-b2e9-2d4e6a2d5f49.mp4`,
}

export const gameStoreSessionGames: GameStoreMockGame[] = [
  { publicId: '615d45e5-a53d-4d07-9868-ba8837b94ac4', title: '半条命', year: '1998', platform: 'PC', coverUrl: `${NAS_API_BASE}/assets/615d45e5-a53d-4d07-9868-ba8837b94ac4/82b4592a-4fdf-4de2-b62a-d373e34f924a.jpg` },
  { publicId: '0693bcf5-d8a0-4ecb-b183-20f4091b1f1b', title: '反恐精英', year: '2000', platform: 'PC', coverUrl: `${NAS_API_BASE}/assets/0693bcf5-d8a0-4ecb-b183-20f4091b1f1b/9691980c-5452-4e36-8878-00a59c5d91ea.png` },
  { publicId: 'e07e358e-10dc-48b2-a83f-dbd16cdd1d81', title: '侠盗猎车手 3', year: '2001', platform: 'PS2 / PC', coverUrl: `${NAS_API_BASE}/assets/e07e358e-10dc-48b2-a83f-dbd16cdd1d81/b16a46d5-3dc7-40dd-9e6e-c3d4e8a821a8.png` },
  { publicId: '6b4406a8-54d3-4c5e-b463-66278df4b824', title: '半条命2', year: '2004', platform: 'PC / Xbox', coverUrl: `${NAS_API_BASE}/assets/6b4406a8-54d3-4c5e-b463-66278df4b824/7f54b19c-6f95-4250-9670-22b1bf85520a.png` },
  { publicId: '4221ffc6-6e86-4055-9639-a71c71f41d23', title: '生化危机5', year: '2009', platform: 'PS3 / Xbox 360 / PC', coverUrl: `${NAS_API_BASE}/assets/4221ffc6-6e86-4055-9639-a71c71f41d23/ae190e95-8d6a-42b3-93cb-cdf27e2e2860.png` },
  { publicId: '95b44dae-e3d3-4e23-b12f-3765c48b974b', title: '求生之路2', year: '2009', platform: 'PC / Xbox 360', coverUrl: `${NAS_API_BASE}/assets/95b44dae-e3d3-4e23-b12f-3765c48b974b/c5b98287-bffa-48c7-92cb-300ddaee78ac.png` },
  { publicId: '4c2fa4cb-4c3b-42bf-8f48-a98586c46cb1', title: '命令与征服：红色警戒3', year: '2009', platform: 'PC / PS3 / Xbox 360', coverUrl: `${NAS_API_BASE}/assets/4c2fa4cb-4c3b-42bf-8f48-a98586c46cb1/7bfba234-1202-4a4c-b1c7-b168a22bfdae.png` },
  { publicId: 'accc79a0-fdef-4cf2-8c8a-30a7254ea544', title: '文明5', year: '2010', platform: 'PC', coverUrl: `${NAS_API_BASE}/assets/accc79a0-fdef-4cf2-8c8a-30a7254ea544/6a628339-7974-48d4-b9b0-01ae829cc361.png` },
  { publicId: '7e73d78a-489d-494f-8987-cb3ffb5ecb77', title: '巫师', year: '2008', platform: 'PC / Xbox 360', coverUrl: `${NAS_API_BASE}/assets/7e73d78a-489d-494f-8987-cb3ffb5ecb77/912dabb5-a924-4a83-abc4-4b714896aa97.png` },
  { publicId: 'e2ad8971-f89f-4754-af74-5a1cd6a81c5b', title: '孤岛惊魂3', year: '2012', platform: 'PC / PS3 / Xbox 360', coverUrl: `${NAS_API_BASE}/assets/e2ad8971-f89f-4754-af74-5a1cd6a81c5b/321a1ef3-f168-42e4-aa24-a182ef0b6545.jpg` },
  { publicId: '56b67626-8492-4f61-8c3a-5b79b9646261', title: '热血无赖', year: '2012', platform: 'PC / PS3 / Xbox 360', coverUrl: `${NAS_API_BASE}/assets/56b67626-8492-4f61-8c3a-5b79b9646261/8aa63159-bfef-4458-8b20-31e0c69743a5.jpg` },
  { publicId: 'dec6b03d-ebd8-4ef1-a732-e820b05923e7', title: '上古卷轴5：天际', year: '2011', platform: 'PC / PS3 / Xbox 360', coverUrl: `${NAS_API_BASE}/assets/dec6b03d-ebd8-4ef1-a732-e820b05923e7/5a43d7c6-1de7-45d0-9799-be4b35735822.jpg` },
  { publicId: '63317972-f522-46c8-873f-e5c2fbd74bb2', title: '刺客信条：黑旗', year: '2013', platform: 'PC / PS3 / Xbox 360', coverUrl: `${NAS_API_BASE}/assets/63317972-f522-46c8-873f-e5c2fbd74bb2/f37d6a63-5764-4c98-b10d-54efd174dcbb.png` },
  { publicId: 'e70536e6-8695-4faf-968f-8f647ff47657', title: '巫师 2：国王刺客', year: '2012', platform: 'PC / Xbox 360', coverUrl: `${NAS_API_BASE}/assets/e70536e6-8695-4faf-968f-8f647ff47657/40fbbcc8-a504-4735-b602-a957b60b137e.png` },
  { publicId: 'ea407b81-ec6a-488c-8934-bf5efd7ad016', title: '极品飞车：最高通缉', year: '2012', platform: 'PC / PS3 / Xbox 360', coverUrl: `${NAS_API_BASE}/assets/ea407b81-ec6a-488c-8934-bf5efd7ad016/ebd7255f-34f3-4f4a-b4bb-ab0bbf09e271.png` },
  { publicId: 'e77f1075-c28b-4d90-990b-5759692183f5', title: '巫师 3：狂猎', year: '2015', platform: 'PC / PS4 / Xbox One', coverUrl: `${NAS_API_BASE}/assets/e77f1075-c28b-4d90-990b-5759692183f5/cceed9bd-fe09-491e-af0f-a1ab8179a54b.png` },
  { publicId: '50f65f81-1e3f-4f66-870f-41b6802854b3', title: '泰坦陨落2', year: '2016', platform: 'PC / PS4 / Xbox One', coverUrl: `${NAS_API_BASE}/assets/50f65f81-1e3f-4f66-870f-41b6802854b3/9f5429e2-6d39-46f1-b506-f8615f38320f.png` },
  { publicId: '176efd32-2c05-4a94-926d-c958bad8d713', title: '荒野大镖客：救赎 2', year: '2018', platform: 'PC / PS4 / Xbox One', coverUrl: `${NAS_API_BASE}/assets/176efd32-2c05-4a94-926d-c958bad8d713/f8d0d049-fa65-42b6-82ea-88871f3f95c0.png` },
  { publicId: '257f3941-368c-45fe-bee9-b2d5b6ff65f0', title: '黑山：起源', year: '2020', platform: 'PC', coverUrl: `${NAS_API_BASE}/assets/257f3941-368c-45fe-bee9-b2d5b6ff65f0/51ed1d27-67e3-4b9d-bbb2-a48809f19034.png` },
  { publicId: '93b97ac2-198a-4de6-a8b5-cb4d4e78c2a1', title: '对马岛之魂', year: '2020', platform: 'PC / PS4 / PS5', coverUrl: `${NAS_API_BASE}/assets/93b97ac2-198a-4de6-a8b5-cb4d4e78c2a1/779bd601-d06f-4ef7-8072-597ba4bfb4d2.jpg` },
]
