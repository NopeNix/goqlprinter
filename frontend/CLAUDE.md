# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## High-Level Architecture

This is a React TypeScript frontend for the Brother QL label printer web application. It provides a web interface for designing and printing labels using Brother QL thermal printers.

### Core Components

- **Multi-page application** built with React Router (`/`, `/qr`)
- **Real-time preview** of labels with adjustable dimensions and content
- **Multiple print modes**: text, QR codes, SVG files, PNG images
- **Printer integration** through REST API endpoints at `/api/*`
- **Settings persistence** via localStorage across sessions

### Technology Stack

- **Frontend**: React 19 with TypeScript
- **Build Tool**: Vite with standard config
- **Styling**: Tailwind CSS with shadcn/ui components
- **UI Components**: Radix UI primitives + shadcn/ui
- **Backend**: Go backend serving REST API on port 8000
- **Packaging**: Docker container with nginx for production

### Key API Endpoints

- `GET /api/printers` - List available printers
- `GET /api/label-sizes` - List supported label sizes
- `POST /api/print` - Print text labels
- `POST /api/print_qr` - Print QR code labels
- `POST /api/print_svg` - Print SVG images
- `POST /api/print_png` - Print PNG images

## Common Commands

```bash
# Development server
npm run dev                    # Runs on http://localhost:5173

# Building
npm run build                  # Build for production (output to ./dist)
npm run preview                # Preview built application

# Code Quality
npm run lint                   # ESLint
npm run lint -- --fix          # Fix ESLint issues
npx tsc --noEmit               # Type checking

# Component Development
npx shadcn@latest add <component>  # Add new shadcn/ui component
```

### Development Setup

1. Backend must be running on `http://localhost:8000` for API calls
2. Proxy configured in `vite.config.ts` for development
3. Settings persist using localStorage keys: "labelSettings" and "qrLabelSettings"

### File Structure

```
src/
├── components/           # Reusable UI components
│   ├── ui/              # shadcn/ui components (button, card, etc.)
│   ├── lib/             # Component utilities
├── pages/               # Route-level components (QRLabelPage.tsx)
├── utils/               # localStorage utilities
└── App.tsx              # Main application with routing
```

### Build Configuration

- **Build output**: `./dist` (shared with backend)
- **Development proxy**: Routes `/api/*` to backend port 8000
- **Tailwind config**: Custom color scheme with CSS variables
- **Vite alias**: `@/` maps to `./src`

### Print Modes

The application supports 4 distinct content types:
1. **Text**: Basic text with font selection and alignment controls
2. **QR Code**: Simple QR code generator with data input
3. **SVG**: Upload and scale vector graphics
4. **PNG**: Upload PNG images for printing

Each mode has specific UI controls and API handling in the print workflow.