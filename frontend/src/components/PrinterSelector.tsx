import React, { useEffect, useState, useCallback, useRef } from 'react';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from './ui/select';
import { Badge } from './ui/badge';
import { Alert, AlertDescription } from './ui/alert';
import { AlertCircle, RefreshCw, CheckCircle } from 'lucide-react';

interface PrinterInfo {
  name: string;
  id: string;
}

interface LabelStatus {
  model_name: string;
  media_type: string;
  media_width: number;
  media_length: number;
  status_type: string;
  phase_type: string;
  errors: string[];
}

interface PrinterStatusResponse {
  status: LabelStatus;
  raw_hex: string;
  raw_bytes: number;
}

interface PrinterSelectorProps {
  value: string | undefined;
  onSelectPrinter: (printer: {id: string, name: string}) => void;
  onDetectLabelSize?: (labelSizeId: string) => void;
  manualOverride?: boolean;
}

const FILE_PRINTER = { id: "file", name: "Print to File (debug)" };
const STORAGE_KEY = "selectedPrinter";
const PRINTER_NAME_KEY = "selectedPrinterName";

// Map printer status to label size ID
function detectLabelSizeFromStatus(status: LabelStatus): string | null {
  const width = status.media_width;
  const length = status.media_length;
  const isContinuous = status.media_type === "Continuous length tape";

  if (isContinuous) {
    // For continuous tape, use width only
    const continuousLabels: Record<number, string> = {
      12: "12",
      18: "18",
      29: "29",
      38: "38",
      50: "50",
      54: "54",
      62: "62",
      102: "102",
      104: "103",
    };
    return continuousLabels[width] || null;
  } else {
    // For die-cut labels, use width x length
    const dieCutLabels: Record<string, string> = {
      "17x54": "17x54",
      "17x87": "17x87",
      "23x23": "23x23",
      "29x42": "29x42",
      "29x90": "29x90",
      "38x90": "39x90",
      "39x48": "39x48",
      "52x29": "52x29",
      "54x29": "54x29",
      "60x87": "60x86",
      "62x29": "62x29",
      "62x100": "62x100",
      "102x51": "102x51",
      "102x153": "102x152",
      "104x164": "103x164",
    };
    const key = `${width}x${length}`;
    return dieCutLabels[key] || null;
  }
}

const PrinterSelector: React.FC<PrinterSelectorProps> = ({ value, onSelectPrinter, onDetectLabelSize, manualOverride }) => {
  const [printers, setPrinters] = useState<PrinterInfo[]>([]);
  const [labelStatus, setLabelStatus] = useState<LabelStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notification, setNotification] = useState<string | null>(null);
  const [refreshDebounced, setRefreshDebounced] = useState(false);
  const hasInitialized = useRef(false);

  // Load saved printer from localStorage (returns both ID and name)
  const loadSavedPrinter = useCallback((): { id: string | null; name: string | null } => {
    try {
      const savedId = localStorage.getItem(STORAGE_KEY);
      const savedName = localStorage.getItem(PRINTER_NAME_KEY);
      return {
        id: savedId ? JSON.parse(savedId) : null,
        name: savedName ? JSON.parse(savedName) : null,
      };
    } catch {
      return { id: null, name: null };
    }
  }, []);

  // Save printer to localStorage
  const savePrinter = useCallback((printerId: string, printerName?: string) => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(printerId));
      if (printerName) {
        localStorage.setItem(PRINTER_NAME_KEY, JSON.stringify(printerName));
      }
    } catch {
      // Silent fail if localStorage is not available
    }
  }, []);

  // Fetch printer list only — no auto-selection logic
  const fetchPrinterList = useCallback(async (): Promise<PrinterInfo[]> => {
    const response = await fetch('/api/printers');
    if (!response.ok) throw new Error('Failed to fetch printers');
    const data = await response.json();
    return data.printers || [];
  }, []);

  // Auto-select printer based on saved state (only called on initial load)
  const autoSelectPrinter = useCallback((printersList: PrinterInfo[]) => {
    const saved = loadSavedPrinter();

    if (saved.id || saved.name) {
      // "file" printer is a valid selection — keep it
      if (saved.id === "file") {
        onSelectPrinter(FILE_PRINTER);
        return;
      }

      // First try to find by exact ID
      const printerById = printersList.find((p: PrinterInfo) => p.id === saved.id);

      if (printerById) {
        onSelectPrinter(printerById);
      } else {
        // ID not found - try to find by model name
        const printerByName = saved.name
          ? printersList.find((p: PrinterInfo) => p.name === saved.name)
          : null;

        if (printerByName) {
          setNotification(`Printer reconnected at new address: ${printerByName.id}`);
          onSelectPrinter(printerByName);
          savePrinter(printerByName.id, printerByName.name);
        } else if (printersList.length > 0) {
          setNotification("Previously selected printer not found. Selected first available.");
          onSelectPrinter(printersList[0]);
          savePrinter(printersList[0].id, printersList[0].name);
        } else {
          setNotification("No printers found. Using file output.");
          onSelectPrinter(FILE_PRINTER);
          savePrinter(FILE_PRINTER.id, FILE_PRINTER.name);
        }
      }
    } else if (printersList.length > 0) {
      onSelectPrinter(printersList[0]);
      savePrinter(printersList[0].id, printersList[0].name);
    } else {
      onSelectPrinter(FILE_PRINTER);
      savePrinter(FILE_PRINTER.id, FILE_PRINTER.name);
    }
  }, [loadSavedPrinter, savePrinter, onSelectPrinter]);

  // Refresh: fetch list + optionally auto-select on first call
  const refreshPrinters = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const printersList = await fetchPrinterList();
      setPrinters(printersList);

      if (!hasInitialized.current) {
        hasInitialized.current = true;
        autoSelectPrinter(printersList);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  }, [fetchPrinterList, autoSelectPrinter]);

  const fetchStatus = useCallback(async () => {
    if (!value || value === 'file') {
      setLabelStatus(null);
      return;
    }

    try {
      const response = await fetch('/api/status', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ printer: value }),
      });

      if (!response.ok) {
        const data = await response.json().catch(() => null);
        const message = data?.message || 'Printer status unavailable';
        setLabelStatus(null);
        setError(response.status === 503 ? message : `Status error: ${message}`);
        return;
      }

      const data: PrinterStatusResponse = await response.json();
      setError(null);
      setLabelStatus(data.status);

      // Auto-detect label size from printer status
      if (onDetectLabelSize && !manualOverride) {
        const detectedLabelId = detectLabelSizeFromStatus(data.status);
        if (detectedLabelId) {
          onDetectLabelSize(detectedLabelId);
        }
      }
    } catch {
      // Network error - printer server not reachable
      setLabelStatus(null);
      setError('Cannot reach printer service');
    }
  }, [value, onDetectLabelSize, manualOverride]);

  // Debounced refresh handler (manual refresh button)
  const handleRefresh = useCallback(() => {
    if (refreshDebounced) return;

    setRefreshDebounced(true);
    setError(null);
    setNotification(null);

    // Manual refresh should re-run auto-selection
    hasInitialized.current = false;
    refreshPrinters().then(() => {
      if (value) {
        fetchStatus();
      }
    });

    setTimeout(() => {
      setRefreshDebounced(false);
    }, 2000);
  }, [refreshDebounced, refreshPrinters, fetchStatus, value]);

  const handleValueChange = useCallback((id: string) => {
    const printer = printers.find(p => p.id === id) || FILE_PRINTER;
    onSelectPrinter(printer);
    savePrinter(printer.id, printer.name);
    setNotification(null);

    // Clear label status when changing printer
    if (id !== value) {
      setLabelStatus(null);
    }
  }, [printers, onSelectPrinter, savePrinter, value]);

  // Initial load
  useEffect(() => {
    refreshPrinters();
  }, [refreshPrinters]);

  // Fetch status when printer is selected
  useEffect(() => {
    if (value && !loading) {
      fetchStatus();
    }
  }, [value, loading, fetchStatus]);

  // Clear notification after 5 seconds
  useEffect(() => {
    if (notification) {
      const timer = setTimeout(() => {
        setNotification(null);
      }, 5000);
      return () => clearTimeout(timer);
    }
  }, [notification]);

  // Visibility/Focus-based refresh - update printer list when user returns to tab
  useEffect(() => {
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        refreshPrinters();
      }
    };

    const handleFocus = () => refreshPrinters();

    document.addEventListener('visibilitychange', handleVisibilityChange);
    window.addEventListener('focus', handleFocus);

    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      window.removeEventListener('focus', handleFocus);
    };
  }, [refreshPrinters]);

  // Light polling - refresh printer list every 30 seconds
  useEffect(() => {
    const interval = setInterval(refreshPrinters, 30000);
    return () => clearInterval(interval);
  }, [refreshPrinters]);

  const selectedPrinterInfo = printers.find(p => p.id === value) ||
    (value === 'file' ? FILE_PRINTER : null);

  if (loading && printers.length === 0) {
    return (
      <div className="space-y-4">
        <div>
          <label className="text-sm font-medium mb-2 block">Select Printer</label>
          <Select value={value} onValueChange={handleValueChange}>
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Select a printer" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="file">Print to File (debug)</SelectItem>
              {printers.map(printer => (
                <SelectItem key={printer.id} value={printer.id}>
                  {printer.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="flex items-center justify-center py-4">
          <RefreshCw className="h-5 w-5 animate-spin mr-2" />
          <span className="text-sm text-muted-foreground">Loading printers...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {error && (
        <Alert variant={error.startsWith('Printer is') || error.startsWith('Printer did') ? "default" : "destructive"}>
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {notification && (
        <Alert variant="default">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{notification}</AlertDescription>
        </Alert>
      )}

      {printers.length === 0 && !loading && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>
            No printers found. Please ensure your USB printer is connected.
          </AlertDescription>
        </Alert>
      )}

      <div>
        <label className="text-sm font-medium mb-2 block">Select Printer</label>
        <Select value={value} onValueChange={handleValueChange}>
          <SelectTrigger className="w-full">
            <SelectValue placeholder="Select a printer" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="file">Print to File (debug)</SelectItem>
            {printers.map(printer => (
              <SelectItem key={printer.id} value={printer.id}>
                {printer.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {selectedPrinterInfo && (
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <CheckCircle className="h-4 w-4 text-green-600" />
            <span className="text-sm font-medium text-green-800">Selected Printer</span>
          </div>
          <div className="text-sm text-green-700 ml-6">
            {selectedPrinterInfo.name} ({selectedPrinterInfo.id})
          </div>
        </div>
      )}

      {/* Printer Status Information - Hide when manual override is enabled */}
      {labelStatus && !manualOverride && (
        <div className="border border-gray-200 rounded-lg p-4 space-y-4">
          <h4 className="text-sm font-medium">Printer Status Information</h4>

          <div>
            <h5 className="text-sm font-medium mb-2">Detected Label:</h5>
            <div className="space-y-2">
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Type:</span>
                <Badge variant="outline">{labelStatus.media_type}</Badge>
              </div>
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Width:</span>
                <Badge variant="outline">{labelStatus.media_width} mm</Badge>
              </div>
              {labelStatus.media_length > 0 && (
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Length:</span>
                  <Badge variant="outline">{labelStatus.media_length} mm</Badge>
                </div>
              )}
            </div>
          </div>

          <div>
            <h5 className="text-sm font-medium mb-2">Printer Status:</h5>
            <div className="space-y-2">
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Model:</span>
                <Badge variant="outline">{labelStatus.model_name}</Badge>
              </div>
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Status:</span>
                <Badge variant={labelStatus.phase_type === "Waiting to receive" ? "secondary" : "outline"}>
                  {labelStatus.phase_type}
                </Badge>
              </div>
              {labelStatus.errors.length > 0 && (
                <div>
                  <span className="text-sm text-muted-foreground">Errors:</span>
                  <div className="mt-1 text-sm text-red-600">
                    {labelStatus.errors.join(", ")}
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {!labelStatus && value && value !== 'file' && !loading && (
        <div className="border border-gray-200 rounded-lg p-4">
          <div className="flex items-center justify-center py-2">
            <span className="text-sm text-muted-foreground">Printer selected. Click Refresh to check status.</span>
          </div>
        </div>
      )}

      <div className="flex justify-end">
        <button
          type="button"
          onClick={handleRefresh}
          disabled={loading || refreshDebounced}
          className="inline-flex items-center rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 border border-input bg-background hover:bg-accent hover:text-accent-foreground px-3 py-1.5"
        >
          <RefreshCw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
          Refresh
        </button>
      </div>
    </div>
  );
};

export default PrinterSelector;
