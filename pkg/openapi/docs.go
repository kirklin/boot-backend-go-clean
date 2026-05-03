package openapi

import "fmt"

// renderScalarHTML returns the HTML page for Scalar API Reference.
// It loads the Scalar library from CDN — no local static files needed.
func renderScalarHTML(title, specURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <title>%s - API Documentation</title>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <style>
    body { margin: 0; padding: 0; }
  </style>
</head>
<body>
  <script
    id="api-reference"
    data-url="%s"
    data-configuration='{"theme":"purple","layout":"modern"}'
  ></script>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`, title, specURL)
}
