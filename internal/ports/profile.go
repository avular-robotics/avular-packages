package ports

import "avular-packages/internal/types"

type ProfileSourcePort interface {
	LoadProfiles(product types.Spec, explicit []string) ([]types.Spec, error)
}
