package bootstrap

// Application holds the core components of the application
type Application struct {
	Config *AppConfig
	// You can add more fields here in the future as needed
}

// NewApplication creates and initializes a new Application instance
func NewApplication() (*Application, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	app := &Application{
		Config: config,
	}

	return app, nil
}

// Initialize performs any necessary setup for the application
func (app *Application) Initialize() error {
	// You can add initialization logic here if needed
	return nil
}

// Shutdown performs any necessary cleanup before the application exits
func (app *Application) Shutdown() {
	// You can add shutdown logic here if needed
}
