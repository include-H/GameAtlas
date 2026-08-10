export interface RequestGenerationGuard {
  isCurrent: () => boolean
}

interface RequestGeneration {
  begin: () => RequestGenerationGuard
  invalidate: () => void
}

export const createRequestGeneration = (): RequestGeneration => {
  let generation = 0

  return {
    begin: () => {
      const requestGeneration = ++generation
      return {
        isCurrent: () => requestGeneration === generation,
      }
    },
    invalidate: () => {
      generation += 1
    },
  }
}
