// @title Label Printer API
// @version 1.0
// @description API for controlling Brother QL label printers
// @BasePath /api
package main

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"goqlprinter/api"
	"goqlprinter/brotherql"
	"goqlprinter/config"
	_ "goqlprinter/docs" // Swagger docs
	"goqlprinter/internal/logging"
	"goqlprinter/services"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

//go:embed all:frontend/dist
var embeddedFiles embed.FS

// Global backend provider instance
// This is initialized once at startup and used throughout the application
var globalBackendProvider brotherql.BackendProvider

// initializeBackendProvider selects and initializes the appropriate backend provider
// based on the configuration setting in config.Cfg.App.Backend
func initializeBackendProvider() brotherql.BackendProvider {
	backend := config.Cfg.App.Backend
	slog.Info("Initializing backend provider", "backend", backend)

	switch backend {
	case "usb":
		// Force USB backend (gousb/libusb)
		slog.Info("Using USB backend (gousb/libusb)")
		return initUSBProvider()

	case "native":
		// Force OS native backend
		slog.Info("Using native OS backend")
		return createNativeProvider()

	case "auto":
		// Try USB first (supports status queries), fallback to native
		slog.Info("Auto mode: trying USB backend first (supports status queries)")
		usbProvider := initUSBProvider()
		if usbProvider != nil {
			printers, err := usbProvider.FindPrinters()
			if err == nil && len(printers) > 0 {
				slog.Info("USB backend found printers, using USB backend", "count", len(printers))
				return usbProvider
			}
			slog.Info("USB backend found no printers, trying native backend", "error", err)
		} else {
			slog.Info("USB backend not available, trying native backend")
		}

		// Fallback to native backend
		nativeProvider := createNativeProvider()
		printers, err := nativeProvider.FindPrinters()
		if err == nil && len(printers) > 0 {
			slog.Info("Native backend found printers, using native backend", "count", len(printers))
			slog.Warn("Note: Native backend does not support printer status queries")
			return nativeProvider
		}

		slog.Warn("No printers found with any backend")
		return nativeProvider // Return native as fallback even if empty

	default:
		slog.Warn("Unknown backend, falling back to auto mode", "backend", backend)
		usbProvider := initUSBProvider()
		if usbProvider != nil {
			printers, err := usbProvider.FindPrinters()
			if err == nil && len(printers) > 0 {
				return usbProvider
			}
		}
		return createNativeProvider()
	}
}

// createNativeProvider creates a platform-specific native backend provider
// Implementation is selected at compile time using build tags
func createNativeProvider() brotherql.BackendProvider {
	return brotherql.NewNativeProvider()
}

func main() {
	// Initialize logging first - with default level to ensure messages are shown
	// We'll re-initialize it later with proper level from environment
	logging.Init("INFO")
	
	// ASCII art banner - always shown
	startupMsg := fmt.Sprintf(`
  ____        _      __  __       _ _             
 |  _ \      | |    |  \/  |     (_) |            
 | |_) | __ _| |__  | \  / | __ _ _| | ___ _ __   
 |  _ < / _' | '_ \ | |\/| |/ _' | | |/ _ \ '_ \  
 | |_) | (_| | |_) || |  | | (_| | | |  __/ | | | 
 |____/ \__,_|_.__(_)_|  |_|\__,_|_|_|\___|_| |_| 
`)
	// Print directly to stdout to ensure it's always visible
	fmt.Println(startupMsg)

	// Load configuration with verbose logging
	slog.Info("Loading configuration...")
	if err := config.LoadConfig(); err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}
	slog.Info("Server configuration loaded successfully",
		"host", config.Cfg.Server.Host,
		"port", config.Cfg.Server.Port,
		"backend", config.Cfg.App.Backend,
		"default_printer", config.Cfg.App.DefaultPrinter,
		"font_dirs", config.Cfg.App.FontDirs)

	// Initialize backend provider based on configuration
	// This validates the backend choice and logs which backend will be used
	globalBackendProvider = initializeBackendProvider()

	// Set the default provider for printer_service.go to use
	// This allows printer discovery to work with different backend implementations
	services.SetDefaultProvider(globalBackendProvider)

	// Initialize default printer using the configured backend provider
	services.InitializeDefaultPrinter(config.Cfg.App.DefaultPrinter)

	// Configure Gin mode first
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		// Show the standard Gin mode warning if GIN_MODE is set
		gin.SetMode(os.Getenv("GIN_MODE"))
	}

	// Re-initialize logging with proper level from environment
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "ERROR"
	}
	logging.Init(logLevel)

	// Configure Gin logging
	if strings.ToUpper(logLevel) == "DEBUG" {
		gin.DefaultWriter = os.Stdout
		gin.DebugPrintRouteFunc = func(httpMethod, absolutePath, handlerName string, nuHandlers int) {
			slog.Debug("route registered", "method", httpMethod, "path", absolutePath, "handler", handlerName)
		}
	} else if strings.ToUpper(logLevel) == "ERROR" {
		gin.DefaultWriter = io.Discard
	} else {
		gin.DefaultWriter = os.Stdout
	}

	// Initial startup logs
	slog.Info("Logging configured successfully")
	slog.Info("Using font directories from config", "font_dirs", config.Cfg.App.FontDirs)

	// Always print the URL information
	fmt.Printf("Brother printer driver is running. Open in browser:\nhttp://%s:%d\n", 
		config.Cfg.Server.Host, config.Cfg.Server.Port)

	// Create router with customized logging
	r := gin.New()
	if logLevel != "ERROR" {
		r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
			return fmt.Sprintf("[%s] %s %s %d %s \"%s\"\n",
				param.TimeStamp.Format("2006-01-02 15:04:05"),
				param.Method,
				param.Path,
				param.StatusCode,
				param.Latency,
				param.ErrorMessage,
			)
		}))
	}
	r.Use(gin.Recovery())

	// Swagger endpoint
	r.GET("/swagger/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL("doc.json"),
		ginSwagger.DefaultModelsExpandDepth(-1),
	))

	// API routes
	apiRoutes := r.Group("/api")
	{
		apiRoutes.GET("/config", api.GetConfig) // Returns ConfigResponse
		apiRoutes.GET("/printers", api.GetPrinters)
		apiRoutes.GET("/label-sizes", api.GetLabelSizes)
		apiRoutes.GET("/label-sizes/:id", api.GetLabelSize)
		apiRoutes.POST("/print", api.PrintLabel)
		apiRoutes.POST("/print_png", api.PrintPNGLabel)
		apiRoutes.POST("/print_png_raw", api.PrintPNGRaw)
		apiRoutes.POST("/print_qr", api.PrintQR)
		apiRoutes.POST("/print_svg", api.PrintSVG)
		apiRoutes.POST("/preview", api.PreviewLabel)
		apiRoutes.GET("/fonts", api.GetFonts)
		apiRoutes.POST("/status", api.GetStatus) // Returns printer status information

		testRoutes := apiRoutes.Group("/test") // Test endpoints for development
		{
			testRoutes.POST("/invalidate", api.TestInvalidate)
			testRoutes.POST("/initialize", api.TestInitialize)
			testRoutes.POST("/feed", api.TestFeed)
			testRoutes.POST("/set_media_and_feed", api.TestSetMediaAndFeed)
		}
	}

	// Serve embedded frontend files

	// 1. Create filesystem for dist directory root
	distFS, err := fs.Sub(embeddedFiles, "frontend/dist")
	if err != nil {
		log.Fatalf("Fatal error: failed to create sub filesystem for dist: %v", err)
	}

	// 2. Create separate filesystem for assets directory
	assetsFS, err := fs.Sub(distFS, "assets")
	if err != nil {
		log.Fatalf("Fatal error: failed to create sub filesystem for assets: %v", err)
	}

	// 3. Serve assets from /assets URL path
	r.StaticFS("/assets", http.FS(assetsFS))

	// 4. Catch-all route for SPA - returns index.html
	r.NoRoute(func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.RequestURI, "/api") {
			c.FileFromFS("/", http.FS(distFS)) // Return root (index.html) from distFS
		}
	})

	// Start server with config values
	listenAddr := fmt.Sprintf("%s:%d", config.Cfg.Server.Host, config.Cfg.Server.Port)
	if err := r.Run(listenAddr); err != nil {
		slog.Error("Failed to run server", "error", err)
		os.Exit(1)
	}
}
