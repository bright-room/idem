package idem

import (
	"context"
	"time"
)

// Storage はべき等キーとレスポンスの永続化を抽象化するインターフェース。
type Storage interface {
	// Get は指定されたキーに対応するキャッシュ済みレスポンスを返す。
	// キーが存在しない場合は nil, nil を返す。
	Get(ctx context.Context, key string) (*Response, error)

	// Set は指定されたキーにレスポンスを保存する。
	// ttl はキャッシュの有効期限を指定する。
	Set(ctx context.Context, key string, res *Response, ttl time.Duration) error
}
