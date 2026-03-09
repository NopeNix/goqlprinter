import React from "react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "./ui/select";
import { ToggleGroup, ToggleGroupItem } from "./ui/toggle-group";
import LabelSizeSelector from "./LabelSizeSelector";
import { RefreshCw, RotateCcw } from "lucide-react";
import type { PrinterInfo } from "../hooks/usePrinterStatus";

interface AdvancedSettingsPanelProps {
  printers: PrinterInfo[];
  selectedPrinter: { id: string; name: string };
  onSelectPrinter: (id: string) => void;
  settingsMode: "auto" | "manual";
  onSettingsModeChange: (mode: "auto" | "manual") => void;
  selectedLabelSize: string;
  onLabelSizeChange: (size: { id: string }) => void;
  selectedOrientation: string;
  onOrientationChange: (value: string) => void;
  printMode: string;
  onResetSettings: () => void;
  loading: boolean;
  refreshDebounced: boolean;
  onRefresh: () => void;
}

const AdvancedSettingsPanel: React.FC<AdvancedSettingsPanelProps> = ({
  printers,
  selectedPrinter,
  onSelectPrinter,
  settingsMode,
  onSettingsModeChange,
  selectedLabelSize,
  onLabelSizeChange,
  selectedOrientation,
  onOrientationChange,
  printMode,
  onResetSettings,
  loading,
  refreshDebounced,
  onRefresh,
}) => {
  const isManual = settingsMode === "manual";

  return (
    <div className="space-y-3 my-4">
      {/* Row 1: Mode toggle + Refresh */}
      <div className="flex items-center gap-2">
        <ToggleGroup
          type="single"
          value={settingsMode}
          onValueChange={onSettingsModeChange}
          className="flex-1"
        >
          <ToggleGroupItem
            value="auto"
            className="flex-1 h-8 text-xs font-medium data-[state=on]:bg-foreground data-[state=on]:text-background"
          >
            Auto
          </ToggleGroupItem>
          <ToggleGroupItem
            value="manual"
            className="flex-1 h-8 text-xs font-medium data-[state=on]:bg-foreground data-[state=on]:text-background"
          >
            Manual
          </ToggleGroupItem>
        </ToggleGroup>
        <button
          type="button"
          onClick={onRefresh}
          disabled={loading || refreshDebounced}
          className="inline-flex items-center justify-center h-8 w-8 rounded-md border border-input bg-background hover:bg-accent transition-colors disabled:opacity-50"
          title="Refresh printers"
        >
          <RefreshCw className={`h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />
        </button>
        <button
          type="button"
          onClick={onResetSettings}
          className="inline-flex items-center justify-center h-8 w-8 rounded-md border border-input bg-background hover:bg-accent text-muted-foreground hover:text-destructive transition-colors"
          title="Reset settings"
        >
          <RotateCcw className="h-3.5 w-3.5" />
        </button>
      </div>

      {/* Row 2: Dropdowns — side by side on desktop, stacked on mobile */}
      <div className={`grid gap-3 ${isManual ? "grid-cols-1 md:grid-cols-3" : "grid-cols-1"}`}>
        <Select value={selectedPrinter?.id} onValueChange={onSelectPrinter}>
          <SelectTrigger className="h-9 text-sm">
            <SelectValue placeholder="Select printer" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="file">Print to File</SelectItem>
            {printers.map((p) => (
              <SelectItem key={p.id} value={p.id}>{p.name}</SelectItem>
            ))}
          </SelectContent>
        </Select>

        {isManual && (
          <>
            <LabelSizeSelector
              value={selectedLabelSize}
              onLabelSizeChange={onLabelSizeChange}
            />
            <Select
              value={selectedOrientation}
              onValueChange={onOrientationChange}
              disabled={printMode === "qr"}
            >
              <SelectTrigger className="h-9 text-sm">
                <SelectValue placeholder="Orientation" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="standard">Standard</SelectItem>
                <SelectItem value="rotated">Rotated</SelectItem>
              </SelectContent>
            </Select>
          </>
        )}
      </div>
    </div>
  );
};

export default AdvancedSettingsPanel;
