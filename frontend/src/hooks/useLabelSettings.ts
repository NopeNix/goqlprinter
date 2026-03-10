import { useReducer, useEffect, useMemo } from "react";
import { loadSettings, saveSettings } from "../utils/localStorageUtils";
import {
  FILE_PRINTER,
  DEFAULT_LABEL_SIZE,
  DEFAULT_DIMENSIONS,
} from "../constants";

// ── Types ──

export type PrintMode = "text" | "qr" | "svg" | "png";

export interface LabelDimensions {
  labelWidth: number;
  labelHeight: number;
  dotsTotalWidth: number;
  dotsTotalHeight: number;
  printableLabelWidth: number;
  printableLabelHeight: number;
}

export interface LabelSettings {
  // Printer
  selectedPrinter: { id: string; name: string };
  selectedLabelSize: string;
  selectedOrientation: string;
  settingsMode: "auto" | "manual";

  // Dimensions
  dimensions: LabelDimensions;

  // Content
  printMode: PrintMode;
  labelText: string;
  selectedFont: string;
  fontSize: number[];
  qrData: string;
  qrScale: number[];
  svgScale: number[];

  // Alignment
  horizontalAlignment: "start" | "center" | "end";
  verticalAlignment: "start" | "center" | "end";
  textRotation: number;

  // Endless tape
  heightMode: "auto" | "manual";
  customHeightMM: number;
}

// ── Actions ──

export type LabelSettingsAction =
  | { type: "SET_PRINTER"; payload: { id: string; name: string } }
  | { type: "SET_LABEL_SIZE"; payload: { id: string; dimensions: LabelDimensions } }
  | { type: "SET_ORIENTATION"; payload: string }
  | { type: "SET_SETTINGS_MODE"; payload: "auto" | "manual" }
  | { type: "SET_PRINT_MODE"; payload: PrintMode }
  | { type: "SET_TEXT"; payload: string }
  | { type: "SET_FONT"; payload: string }
  | { type: "SET_FONT_SIZE"; payload: number[] }
  | { type: "SET_QR_DATA"; payload: string }
  | { type: "SET_QR_SCALE"; payload: number[] }
  | { type: "SET_SVG_SCALE"; payload: number[] }
  | { type: "SET_HORIZONTAL_ALIGNMENT"; payload: "start" | "center" | "end" }
  | { type: "SET_VERTICAL_ALIGNMENT"; payload: "start" | "center" | "end" }
  | { type: "SET_TEXT_ROTATION"; payload: number }
  | { type: "SET_HEIGHT_MODE"; payload: "auto" | "manual" }
  | { type: "SET_CUSTOM_HEIGHT_MM"; payload: number }
  | { type: "RESET" };

// ── Defaults ──

export const DEFAULT_SETTINGS: LabelSettings = {
  selectedPrinter: { ...FILE_PRINTER },
  selectedLabelSize: DEFAULT_LABEL_SIZE,
  selectedOrientation: "standard",
  settingsMode: "auto",
  dimensions: { ...DEFAULT_DIMENSIONS },
  printMode: "text",
  labelText: "Hello, World!",
  selectedFont: "",
  fontSize: [20],
  qrData: "",
  qrScale: [100],
  svgScale: [100],
  horizontalAlignment: "center",
  verticalAlignment: "center",
  textRotation: 0,
  heightMode: "auto",
  customHeightMM: 0,
};

// ── Migration ──

function migrateFromOldFormat(saved: Record<string, unknown>): LabelSettings {
  // Detect old format: has flat dimension keys like labelWidth
  if ("labelWidth" in saved && !("dimensions" in saved)) {
    return {
      selectedPrinter:
        (saved.selectedPrinter as { id: string; name: string }) ||
        DEFAULT_SETTINGS.selectedPrinter,
      selectedLabelSize:
        (saved.selectedLabelSize as string) || DEFAULT_SETTINGS.selectedLabelSize,
      selectedOrientation:
        (saved.selectedOrientation as string) || DEFAULT_SETTINGS.selectedOrientation,
      settingsMode:
        (saved.settingsMode as "auto" | "manual") || DEFAULT_SETTINGS.settingsMode,
      dimensions: {
        labelWidth: (saved.labelWidth as number) || DEFAULT_DIMENSIONS.labelWidth,
        labelHeight: (saved.labelHeight as number) ?? DEFAULT_DIMENSIONS.labelHeight,
        dotsTotalWidth:
          (saved.dotsTotalWidth as number) || DEFAULT_DIMENSIONS.dotsTotalWidth,
        dotsTotalHeight:
          (saved.dotsTotalHeight as number) ?? DEFAULT_DIMENSIONS.dotsTotalHeight,
        printableLabelWidth:
          (saved.printableLabelWidth as number) ||
          DEFAULT_DIMENSIONS.printableLabelWidth,
        printableLabelHeight:
          (saved.printableLabelHeight as number) ??
          DEFAULT_DIMENSIONS.printableLabelHeight,
      },
      printMode: (saved.printMode as PrintMode) || DEFAULT_SETTINGS.printMode,
      labelText: (saved.labelText as string) ?? DEFAULT_SETTINGS.labelText,
      selectedFont: (saved.selectedFont as string) ?? DEFAULT_SETTINGS.selectedFont,
      fontSize: (saved.fontSize as number[]) || DEFAULT_SETTINGS.fontSize,
      qrData: (saved.qrData as string) ?? DEFAULT_SETTINGS.qrData,
      qrScale: (saved.qrScale as number[]) || DEFAULT_SETTINGS.qrScale,
      svgScale: (saved.svgScale as number[]) || DEFAULT_SETTINGS.svgScale,
      horizontalAlignment:
        (saved.horizontalAlignment as "start" | "center" | "end") ||
        DEFAULT_SETTINGS.horizontalAlignment,
      verticalAlignment:
        (saved.verticalAlignment as "start" | "center" | "end") ||
        DEFAULT_SETTINGS.verticalAlignment,
      textRotation:
        (saved.textRotation as number) ?? DEFAULT_SETTINGS.textRotation,
      heightMode:
        (saved.heightMode as "auto" | "manual") || DEFAULT_SETTINGS.heightMode,
      customHeightMM:
        (saved.customHeightMM as number) ?? DEFAULT_SETTINGS.customHeightMM,
    };
  }

  // New format — merge with defaults for any missing keys
  return { ...DEFAULT_SETTINGS, ...(saved as Partial<LabelSettings>) };
}

function loadInitialState(): LabelSettings {
  const saved = loadSettings();
  if (!saved) return { ...DEFAULT_SETTINGS };
  return migrateFromOldFormat(saved);
}

// ── Reducer ──

function labelSettingsReducer(
  state: LabelSettings,
  action: LabelSettingsAction,
): LabelSettings {
  switch (action.type) {
    case "SET_PRINTER":
      return { ...state, selectedPrinter: action.payload };
    case "SET_LABEL_SIZE":
      return {
        ...state,
        selectedLabelSize: action.payload.id,
        dimensions: action.payload.dimensions,
      };
    case "SET_ORIENTATION":
      return { ...state, selectedOrientation: action.payload };
    case "SET_SETTINGS_MODE":
      return { ...state, settingsMode: action.payload };
    case "SET_PRINT_MODE":
      return { ...state, printMode: action.payload };
    case "SET_TEXT":
      return { ...state, labelText: action.payload };
    case "SET_FONT":
      return { ...state, selectedFont: action.payload };
    case "SET_FONT_SIZE":
      return { ...state, fontSize: action.payload };
    case "SET_QR_DATA":
      return { ...state, qrData: action.payload };
    case "SET_QR_SCALE":
      return { ...state, qrScale: action.payload };
    case "SET_SVG_SCALE":
      return { ...state, svgScale: action.payload };
    case "SET_HORIZONTAL_ALIGNMENT":
      return { ...state, horizontalAlignment: action.payload };
    case "SET_VERTICAL_ALIGNMENT":
      return { ...state, verticalAlignment: action.payload };
    case "SET_TEXT_ROTATION":
      return { ...state, textRotation: action.payload };
    case "SET_HEIGHT_MODE":
      return { ...state, heightMode: action.payload };
    case "SET_CUSTOM_HEIGHT_MM":
      return { ...state, customHeightMM: action.payload };
    case "RESET":
      return { ...DEFAULT_SETTINGS };
    default:
      return state;
  }
}

// ── Hook ──

export interface UseLabelSettingsReturn {
  settings: LabelSettings;
  dispatch: React.Dispatch<LabelSettingsAction>;
  isEndlessTape: boolean;
}

export function useLabelSettings(): UseLabelSettingsReturn {
  const [settings, dispatch] = useReducer(
    labelSettingsReducer,
    undefined,
    loadInitialState,
  );

  // Auto-save to localStorage on every state change
  useEffect(() => {
    saveSettings(settings);
  }, [settings]);

  const isEndlessTape = useMemo(
    () =>
      settings.dimensions.labelHeight === 0 ||
      settings.dimensions.printableLabelHeight === 0,
    [settings.dimensions.labelHeight, settings.dimensions.printableLabelHeight],
  );

  return { settings, dispatch, isEndlessTape };
}
