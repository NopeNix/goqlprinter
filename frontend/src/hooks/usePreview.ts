import { useState, useEffect, useRef, useCallback } from 'react';
import { printApi } from '../api/endpoints';

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
  // Endless tape custom height
  customHeightMM?: number;
  // Control
  enabled?: boolean; // default true, allows disabling preview fetching
}

interface UsePreviewResult {
  previewUrl: string | null;
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
  custom_height_mm?: number;
  [key: string]: unknown;
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
    customHeightMM,
    enabled = true,
  } = params;

  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
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

      // Add custom height for endless tape
      if (customHeightMM && customHeightMM > 0) {
        requestBody.custom_height_mm = customHeightMM;
      }

      // Add SVG fields if present
      if (svgData) {
        requestBody.svg_data = svgData;
      }
      if (svgScale !== undefined) {
        requestBody.svg_scale = svgScale;
      }

      const data = await printApi.preview(requestBody, signal);

      // Only update state if request wasn't aborted
      if (!signal.aborted) {
        setPreviewUrl(data.image);
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
    customHeightMM,
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
    isLoading,
    error,
  };
}

export default usePreview;
