package ports

import "avular-packages/internal/types"

type ProductSpecPort interface {
	LoadProduct(path string) (types.Spec, error)
}

type ProfileSpecPort interface {
	LoadProfile(path string) (types.Spec, error)
}
