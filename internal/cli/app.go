package cli

import "avular-packages/internal/app"

func newAppService() app.Service {
	return app.NewService()
}
