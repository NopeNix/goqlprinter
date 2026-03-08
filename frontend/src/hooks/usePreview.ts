import { useState, useEffect, useRef, useCallback } from 'react';

interface UsePreviewParams {
  text: string;
  labelSize: string;
  fontFamily: string;
  fontSize: number;
  orientation: string;
  horizontalAlignment: string;
  verticalAlignment: string;
  textRotation: number;
  // SVG optional
  svgData?: string | null;
  svgScale?: number;
  svgHorizontalAlignment?: string;
  svgVerticalAlignment?: string;
  // Control
  enabled?: boolean; // default true, allows disabling preview fetching
}

interface UsePreviewResult {
  previewUrl: string | null;
  previewDimensions: { width: number; height: number } | null;
  isLoading: boolean;
  error: string | null;
}

interface PreviewRequest {
  text: string;
  label_size: string;
  font_family: string;
  font_size: number;
  orientation: string;
  horizontal_alignment: string;
  vertical_alignment: string;
  text_rotation: number;
  svg_data?: string;
  svg_scale?: number;
  svg_horizontal_alignment?: string;
  svg_vertical_alignment?: string;
}

interface PreviewResponse {
  image: string; // "data:image/png;base64,..."
  width: number;
  height: number;
  printable_width: number;
  printable_height: number;
}

const DEBOUNCE_DELAY = 250; // ms

/**
 * Hook to fetch preview images from the backend /api/preview endpoint.
 *
 * Features:
 * - 250ms debouncing on parameter changes
 * - AbortController to cancel in-flight requests
 * - Proper loading and error state management
 * - Cleanup on unmount
 */
export function usePreview(params: UsePreviewParams): UsePreviewResult {
  const {
    text,
    labelSize,
    fontFamily,
    fontSize,
    orientation,
    horizontalAlignment,
    verticalAlignment,
    textRotation,
    svgData,
    svgScale,
    svgHorizontalAlignment,
    svgVerticalAlignment,
    enabled = true,
  } = params;

  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [previewDimensions, setPreviewDimensions] = useState<{ width: number; height: number } | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  // Refs for cleanup
  const abortControllerRef = useRef<AbortController | null>(null);
  const debounceTimerRef = useRef<NodeJS.Timeout | null>(null);

  /**
   * Fetch preview from backend
   */
  const fetchPreview = useCallback(async (signal: AbortSignal) => {
    try {
      setIsLoading(true);
      setError(null);

      const requestBody: PreviewRequest = {
        text,
        label_size: labelSize,
        font_family: fontFamily,
        font_size: fontSize,
        orientation,
        horizontal_alignment: horizontalAlignment,
        vertical_alignment: verticalAlignment,
        text_rotation: textRotation,
      };

      // Add SVG fields if present
      if (svgData) {
        requestBody.svg_data = svgData;
      }
      if (svgScale !== undefined) {
        requestBody.svg_scale = svgScale;
      }
      if (svgHorizontalAlignment) {
        requestBody.svg_horizontal_alignment = svgHorizontalAlignment;
      }
      if (svgVerticalAlignment) {
        requestBody.svg_vertical_alignment = svgVerticalAlignment;
      }

      const response = await fetch('/api/preview', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(requestBody),
        signal,
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: 'Unknown error' }));
        throw new Error(errorData.error || `HTTP ${response.status}`);
      }

      const data: PreviewResponse = await response.json();

      // Only update state if request wasn't aborted
      if (!signal.aborted) {
        setPreviewUrl(data.image);
        setPreviewDimensions({
          width: data.width,
          height: data.height,
        });
        setIsLoading(false);
      }
    } catch (err) {
      // Don't set error state for aborted requests
      if (err instanceof Error && err.name === 'AbortError') {
        return;
      }

      if (!signal.aborted) {
        const errorMessage = err instanceof Error ? err.message : 'Failed to fetch preview';
        setError(errorMessage);
        setPreviewUrl(null);
        setPreviewDimensions(null);
        setIsLoading(false);
      }
    }
  }, [
    text,
    labelSize,
    fontFamily,
    fontSize,
    orientation,
    horizontalAlignment,
    verticalAlignment,
    textRotation,
    svgData,
    svgScale,
    svgHorizontalAlignment,
    svgVerticalAlignment,
  ]);

  /**
   * Effect to handle debounced preview fetching
   */
  useEffect(() => {
    // Don't fetch if disabled
    if (!enabled) {
      setIsLoading(false);
      return;
    }

    // Clear any existing debounce timer
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }

    // Abort any in-flight request
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }

    // Create new abort controller for this request
    const abortController = new AbortController();
    abortControllerRef.current = abortController;

    // Set up debounced fetch
    debounceTimerRef.current = setTimeout(() => {
      fetchPreview(abortController.signal);
    }, DEBOUNCE_DELAY);

    // Cleanup function
    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
    };
  }, [enabled, fetchPreview]);

  return {
    previewUrl,
    previewDimensions,
    isLoading,
    error,
  };
}

export default usePreview;
