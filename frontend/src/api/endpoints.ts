import { apiGet, apiPost } from "./client";

// Types

interface PrinterListResponse {
  printers: { id: string; name: string }[];
}

interface LabelSizeResponse {
  id: string;
  tape_size_width: number;
  tape_size_height: number;
  dots_total_width: number;
  dots_total_height: number;
  dots_printable_width: number;
  dots_printable_height: number;
}

interface LabelSizeListResponse {
  label_sizes: { id: string; name: string }[];
}

interface PrinterStatusResponse {
  status: {
    model_name: string;
    media_type: string;
    media_width: number;
    media_length: number;
    status_type: string;
    phase_type: string;
    errors: string[];
  };
  raw_hex: string;
  raw_bytes: number;
}

interface PreviewResponse {
  image: string;
  width: number;
  height: number;
  printable_width: number;
  printable_height: number;
}

interface FontListResponse {
  fonts: string[];
}

export const printerApi = {
  list: (signal?: AbortSignal) =>
    apiGet<PrinterListResponse>("/api/printers", signal),
  status: (body: { printer: string }, signal?: AbortSignal) =>
    apiPost<PrinterStatusResponse>("/api/status", body, signal),
};

export const labelApi = {
  sizes: (signal?: AbortSignal) =>
    apiGet<LabelSizeListResponse>("/api/label-sizes", signal),
  size: (id: string, signal?: AbortSignal) =>
    apiGet<LabelSizeResponse>(`/api/label-sizes/${id}`, signal),
};

export const printApi = {
  text: (body: Record<string, unknown>) =>
    apiPost<Record<string, unknown>>("/api/print", body),
  qr: (body: Record<string, unknown>) =>
    apiPost<Record<string, unknown>>("/api/print_qr", body),
  svg: (body: Record<string, unknown>) =>
    apiPost<Record<string, unknown>>("/api/print_svg", body),
  png: (body: Record<string, unknown>) =>
    apiPost<Record<string, unknown>>("/api/print_png", body),
  preview: (body: Record<string, unknown>, signal?: AbortSignal) =>
    apiPost<PreviewResponse>("/api/preview", body, signal),
};

export const fontApi = {
  list: (signal?: AbortSignal) =>
    apiGet<FontListResponse>("/api/fonts", signal),
};
