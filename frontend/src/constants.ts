export const DOTS_PER_MM = 11.81;

export const FILE_PRINTER = { id: "file", name: "Print to File" } as const;

export const DEFAULT_LABEL_SIZE = "62x29";

export const DEFAULT_DIMENSIONS = {
  labelWidth: 62,
  labelHeight: 29,
  dotsTotalWidth: 732,
  dotsTotalHeight: 341,
  printableLabelWidth: 696,
  printableLabelHeight: 271,
} as const;
