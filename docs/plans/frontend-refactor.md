# Frontend Refactoring Plan

Status: **Draft**
Created: 2026-03-09

## Problem

App.tsx on 876-rivinen monoliitti jossa 25+ useState-kutsua. QRLabelPage kopioi saman logiikan. Ei API-kerrosta, ei testejä, props drilling kaikkialla.

## Goals

1. Pilkkoa App.tsx hallittaviin osiin
2. Poistaa koodiduplikaatio QRLabelPage vs MainApp
3. Yhdenmukaistaa API-kutsut
4. Mahdollistaa testaus

## Non-Goals

- UI:n visuaalinen muuttaminen
- Backend API:n muutokset
- Uusien ominaisuuksien lisääminen

---

## Phase 1: useLabelSettings hook + useReducer

**Tavoite:** Siirtää 25+ useState-kutsua yhteen hookiin ryhmiteltynä.

### Uusi tiedosto: `src/hooks/useLabelSettings.ts`

```typescript
interface LabelDimensions {
  labelWidth: number;
  labelHeight: number;
  dotsTotalWidth: number;
  dotsTotalHeight: number;
  printableLabelWidth: number;
  printableLabelHeight: number;
}

interface AlignmentSettings {
  horizontalAlignment: 'start' | 'center' | 'end';
  verticalAlignment: 'start' | 'center' | 'end';
  textRotation: number;
  svgHorizontalAlignment: 'start' | 'center' | 'end';
  svgVerticalAlignment: 'start' | 'center' | 'end';
}

interface LabelSettings {
  // Printer
  selectedPrinter: PrinterInfo;
  selectedLabelSize: string;
  selectedOrientation: string;
  settingsMode: 'auto' | 'manual';

  // Dimensions
  dimensions: LabelDimensions;

  // Content
  printMode: PrintMode;
  labelText: string;
  selectedFont: string;
  fontSize: number[];
  qrData: string;
  svgScale: number[];

  // Alignment
  alignment: AlignmentSettings;

  // Endless tape
  heightMode: 'auto' | 'manual';
  customHeightMM: number;
}

type LabelAction =
  | { type: 'SET_PRINTER'; payload: PrinterInfo }
  | { type: 'SET_LABEL_SIZE'; payload: { id: string; dimensions: LabelDimensions } }
  | { type: 'SET_ALIGNMENT'; payload: Partial<AlignmentSettings> }
  | { type: 'SET_CONTENT'; payload: Partial<Pick<LabelSettings, 'labelText' | 'fontSize' | 'selectedFont' | 'qrData'>> }
  | { type: 'SET_PRINT_MODE'; payload: PrintMode }
  | { type: 'SET_ORIENTATION'; payload: string }
  | { type: 'SET_HEIGHT_MODE'; payload: { mode: 'auto' | 'manual'; customMM?: number } }
  | { type: 'RESET' };
```

### Muutokset App.tsx:ssä

- Korvaa 25 useState-kutsua yhdellä: `const [settings, dispatch] = useLabelSettings()`
- Poista manuaalinen useEffect-synkronointi localStorageen (hook hoitaa)
- Johdettavat arvot (isEndlessTape, printableAreaLabel) hookista tai useMemo:lla

### Vaiheet

1. Luo `useLabelSettings.ts` reducerilla ja localStorage-synkronoinnilla
2. Luo `src/constants.ts` maagisillle arvoille (DOTS_PER_MM, FILE_PRINTER, DEFAULT_LABEL_SIZE)
3. Korvaa App.tsx:n useState-kutsut hookin käytöllä
4. Varmista tyyppitarkistus: `npx tsc --noEmit`
5. Testaa manuaalisesti: asetukset tallentuvat, latautuvat, resetoituvat

### Riskit

- Reducer-migraatio voi rikkoa yksittäisiä setteriä jotka välitetään propsina
- localStorage-skeema muuttuu -> tarvitaan migraatio vanhasta muodosta

---

## Phase 2: usePrintJob hook

**Tavoite:** Yhteinen tulostuslogiikka App.tsx:lle ja QRLabelPage:lle.

### Uusi tiedosto: `src/hooks/usePrintJob.ts`

```typescript
interface PrintJobOptions {
  settings: LabelSettings;
  svgData: string | null;
  pngFile: File | null;
}

interface PrintJobResult {
  handlePrint: () => Promise<void>;
  isPrinting: boolean;
  printerError: string | null;
  showDisconnected: boolean;
  recovery: {
    isRecovering: boolean;
    retry: () => void;
    cancel: () => void;
  };
}

function usePrintJob(options: PrintJobOptions): PrintJobResult
```

### Muutokset

- Siirrä App.tsx handlePrint() + virheenkäsittely + recovery hookiin
- QRLabelPage käyttää samaa hookia
- Endpoint-valinta printMode:n perusteella hookin sisällä

### Vaiheet

1. Luo `usePrintJob.ts`
2. Siirrä handlePrint-logiikka App.tsx -> hook
3. Siirrä usePrinterRecovery-integraatio hookin sisälle
4. Päivitä App.tsx käyttämään hookia
5. Päivitä QRLabelPage käyttämään hookia
6. Poista QRLabelPage:n duplikaattikoodi

---

## Phase 3: API-kerros

**Tavoite:** Keskitetty fetch-wrapper virheenkäsittelyineen.

### Uusi tiedosto: `src/api/client.ts`

```typescript
class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,     // 'USB_DEVICE_NOT_FOUND' | 'VALIDATION_ERROR' | ...
    message: string
  ) {
    super(message);
  }
}

async function apiGet<T>(path: string, signal?: AbortSignal): Promise<T>
async function apiPost<T>(path: string, body: unknown, signal?: AbortSignal): Promise<T>
```

### Uusi tiedosto: `src/api/endpoints.ts`

```typescript
export const printerApi = {
  list: () => apiGet<PrinterInfo[]>('/api/printers'),
  status: (printer: string) => apiPost<PrinterStatus>('/api/status', { printer }),
};

export const labelApi = {
  sizes: () => apiGet<LabelSize[]>('/api/label-sizes'),
  size: (id: string) => apiGet<LabelSize>(`/api/label-sizes/${id}`),
};

export const printApi = {
  text: (payload: TextPrintPayload) => apiPost('/api/print', payload),
  qr: (payload: QRPrintPayload) => apiPost('/api/print_qr', payload),
  svg: (payload: SVGPrintPayload) => apiPost('/api/print_svg', payload),
  png: (payload: PNGPrintPayload) => apiPost('/api/print_png', payload),
};
```

### Muutokset

- Korvaa kaikki suorat fetch-kutsut api-funktioilla
- Virheentunnistus koodilla eikä merkkijonolla: `if (error.code === 'USB_DEVICE_NOT_FOUND')`
- **Edellyttää backend-muutoksen:** Lisää `error_code` kenttä API-vastauksiin

### Vaiheet

1. Luo `src/api/client.ts` ja `src/api/endpoints.ts`
2. Päivitä usePreview.ts käyttämään api-clientia
3. Päivitä usePrinterStatus.ts
4. Päivitä usePrintJob.ts (phase 2:sta)
5. Päivitä FontSelector.tsx ja LabelSizeSelector.tsx
6. (Backend) Lisää error_code JSON-vastauksiin

---

## Phase 4: Pienet korjaukset

Nämä voi tehdä missä vaiheessa tahansa:

| Korjaus | Tiedosto | Kuvaus |
|---------|----------|--------|
| Vakiot | `src/constants.ts` | DOTS_PER_MM, FILE_PRINTER, DEFAULT_LABEL_SIZE, PREVIEW_SCALE_FACTOR |
| Dead code | `usePreview.ts` | Poista käyttämätön previewDimensions |
| Console.log | `App.tsx:439` | Poista "Sending print request" tuotannosta |
| useMemo | `LabelPreview.tsx` | Memoisoi style-objektit |
| Cleanup | `usePrinterRecovery.ts:~199` | useState -> useEffect cleanupille |

---

## Toteutusjärjestys

```
Phase 1 (useLabelSettings)
    │
    ├── Phase 4 (pienet korjaukset, rinnakkain)
    │
    ▼
Phase 2 (usePrintJob)
    │
    ▼
Phase 3 (API-kerros)
```

Phase 1 on edellytys Phase 2:lle koska usePrintJob tarvitsee LabelSettings-tyypin.
Phase 3 voi tehdä Phase 2:n jälkeen tai itsenäisesti, mutta vaatii backend-muutoksen.
Phase 4 on itsenäinen ja voi tehdä milloin vain.

## Estimoitu laajuus

- Phase 1: Suurin yksittäinen muutos, koskee App.tsx:n ydintä
- Phase 2: Keskikokoinen, vaatii QRLabelPage:n uudelleenkirjoituksen
- Phase 3: Pieni mutta koskee montaa tiedostoa + backend
- Phase 4: Triviaali, yksittäisiä korjauksia
