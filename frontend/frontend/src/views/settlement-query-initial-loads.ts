export function runSettlementQueryInitialLoads(loaders: Array<() => Promise<unknown>>): Promise<unknown[]> {
  return Promise.all(loaders.map((load) => load()))
}
