import { Printer, Wifi, WifiOff, AlertCircle, ChevronDown, RefreshCw } from "lucide-react";

export type PrinterStatusKind = "ready" | "busy" | "error" | "offline" | "file";

interface PrinterStatusBarProps {
  printerName: string;
  labelSize: string;
  labelWidth: number;
  labelHeight: number;
  status?: PrinterStatusKind;
  statusDetail?: string;
  expanded?: boolean;
  onClick?: () => void;
  onRefresh?: () => void;
  refreshLoading?: boolean;
  refreshDebounced?: boolean;
}

function formatLabelSize(width: number, height: number): string {
  if (height === 0) return `${width}mm endless`;
  return `${width}×${height}mm`;
}

const statusConfig: Record<PrinterStatusKind, {
  icon: typeof Wifi;
  className: string;
  dotClassName: string;
  label: string;
}> = {
  ready: {
    icon: Wifi,
    className: "text-emerald-600 dark:text-emerald-400",
    dotClassName: "bg-emerald-500",
    label: "Ready",
  },
  busy: {
    icon: Wifi,
    className: "text-amber-600 dark:text-amber-400",
    dotClassName: "bg-amber-500",
    label: "Busy",
  },
  error: {
    icon: AlertCircle,
    className: "text-red-600 dark:text-red-400",
    dotClassName: "bg-red-500",
    label: "Error",
  },
  offline: {
    icon: WifiOff,
    className: "text-red-600 dark:text-red-400",
    dotClassName: "bg-red-500",
    label: "Offline",
  },
  file: {
    icon: WifiOff,
    className: "text-zinc-500 dark:text-zinc-400",
    dotClassName: "bg-zinc-400",
    label: "File",
  },
};

export default function PrinterStatusBar({
  printerName,
  labelWidth,
  labelHeight,
  status = "ready",
  statusDetail,
  expanded = false,
  onClick,
  onRefresh,
  refreshLoading = false,
  refreshDebounced = false,
}: PrinterStatusBarProps) {
  const cfg = statusConfig[status];
  const pillLabel = statusDetail || cfg.label;

  return (
    <header
      className={`border-b bg-background sticky top-0 z-40 ${
        onClick ? "cursor-pointer select-none" : ""
      }`}
      onClick={onClick}
    >
      <div className="container flex items-center h-14 px-4 gap-3">
        {/* App title */}
        <h1 className="text-sm font-semibold tracking-tight whitespace-nowrap">
          QL Label Designer
        </h1>

        {/* Separator */}
        <div className="h-4 w-px bg-border flex-shrink-0" />

        {/* Printer info — clickable area */}
        <div className="flex items-center gap-2 min-w-0 flex-1">
          <Printer className="h-3.5 w-3.5 text-muted-foreground flex-shrink-0" />
          <span className="text-sm text-muted-foreground truncate">
            {printerName}
          </span>
          <span className="text-muted-foreground/40 flex-shrink-0">·</span>
          <span className="text-sm text-muted-foreground whitespace-nowrap flex-shrink-0">
            {formatLabelSize(labelWidth, labelHeight)}
          </span>
        </div>

        {/* Refresh + Status + expand indicator */}
        <div className="flex items-center gap-1.5 flex-shrink-0">
          {onRefresh && (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                onRefresh();
              }}
              disabled={refreshLoading || refreshDebounced}
              className="inline-flex items-center justify-center h-6 w-6 rounded-md hover:bg-accent transition-colors disabled:opacity-50"
              title="Refresh printers"
            >
              <RefreshCw className={`h-3 w-3 text-muted-foreground ${refreshLoading ? "animate-spin" : ""}`} />
            </button>
          )}
          <span className={`inline-flex items-center gap-1.5 text-xs font-medium ${cfg.className}`}>
            <span className={`h-1.5 w-1.5 rounded-full ${cfg.dotClassName}`} />
            {pillLabel}
          </span>
          {onClick && (
            <ChevronDown
              className={`h-3.5 w-3.5 text-muted-foreground transition-transform duration-200 ${
                expanded ? "rotate-180" : ""
              }`}
            />
          )}
        </div>
      </div>
    </header>
  );
}
