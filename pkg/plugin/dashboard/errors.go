package dashboard

import "errors"

var (
	ErrJavaScriptReturnedNoPanels = errors.New("javascript returned no panels")
	ErrDashboardHTTPError         = errors.New("dashboard request does not return 200 OK")
	ErrImageRendererHTTPError     = errors.New("imager renderer request does not return 200 OK")
	ErrEmptyBlobURL               = errors.New("empty blob URL")
	ErrEmptyCSVData               = errors.New("empty csv data")
)
