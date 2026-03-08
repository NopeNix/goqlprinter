// @title Label Printer API
// @version 1.0
// @description API for controlling Brother QL label printers
// @BasePath /api
package main

import (
	"goqlprinter/api"
	"goqlprinter/brotherql"
	_ "goqlprinter/docs" // Swagger docs
	"goqlprinter/config"
	"goqlprinter/logger"
	"goqlprinter/services"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"io"
	"os"
	"strings"

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
	logger.Info("Initializing backend provider: %s", backend)

	switch backend {
	case "usb":
		// Force USB backend (gousb/libusb)
		logger.Info("Using USB backend (gousb/libusb)")
		return initUSBProvider()

	case "native":
		// Force OS native backend
		logger.Info("Using native OS backend")
		return createNativeProvider()

	case "auto":
		// Try USB first (supports status queries), fallback to native
		logger.Info("Auto mode: trying USB backend first (supports status queries)")
		usbProvider := initUSBProvider()
		if usbProvider != nil {
			printers, err := usbProvider.FindPrinters()
			if err == nil && len(printers) > 0 {
				logger.Info("USB backend found %d printer(s), using USB backend", len(printers))
				return usbProvider
			}
			logger.Info("USB backend found no printers (err=%v), trying native backend", err)
		} else {
			logger.Info("USB backend not available, trying native backend")
		}

		// Fallback to native backend
		nativeProvider := createNativeProvider()
		printers, err := nativeProvider.FindPrinters()
		if err == nil && len(printers) > 0 {
			logger.Info("Native backend found %d printer(s), using native backend", len(printers))
			logger.Warning("Note: Native backend does not support printer status queries")
			return nativeProvider
		}

		logger.Warning("No printers found with any backend")
		return nativeProvider // Return native as fallback even if empty

	default:
		logger.Warning("Unknown backend '%s', falling back to auto mode", backend)
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
	logger.Init("INFO")
	
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
	logger.Info("Loading configuration...")
	if err := config.LoadConfig(); err != nil {
		logger.Error("Failed to load configuration: %v", err)
		os.Exit(1)
	}
	logger.Info("Server configuration loaded successfully:")
	logger.Info("- Host: %s", config.Cfg.Server.Host)
	logger.Info("- Port: %d", config.Cfg.Server.Port)
	logger.Info("- Backend: %s", config.Cfg.App.Backend)
	logger.Info("- Default Printer: %s", config.Cfg.App.DefaultPrinter)
	logger.Info("- Font directories: %v", config.Cfg.App.FontDirs)

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
	if logLevel == "" && gin.Mode() == gin.ReleaseMode {
		logLevel = "ERROR" // Default to ERROR level in release mode
	}
	logger.Init(logLevel)

	// Configure Gin logging
	switch logger.GetLevel() {
	case logger.DEBUG:
		gin.DefaultWriter = os.Stdout
		gin.DebugPrintRouteFunc = func(httpMethod, absolutePath, handlerName string, nuHandlers int) {
			logger.Debug("%-6s %-25s --> %s (%d handlers)", httpMethod, absolutePath, handlerName, nuHandlers)
		}
	case logger.INFO, logger.WARNING:
		gin.DefaultWriter = os.Stdout
	case logger.ERROR:
		gin.DefaultWriter = io.Discard
	}

	// Initial startup logs
	logger.Info("Logging configured successfully")
	logger.Info("Using font directories from config: %v", config.Cfg.App.FontDirs)

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
		logger.Error("Failed to run server: %v", err)
		os.Exit(1)
	}
}
