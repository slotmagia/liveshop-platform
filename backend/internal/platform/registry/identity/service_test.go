package identity

import (
	"bytes"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lvtuopen-ai/kernel-go/accessidentity"
)

func TestPlatformLoginDoesNotAcceptTenantAsRequiredInput(t *testing.T) {
	if !validLogin(LoginInput{Realm: accessidentity.RealmPlatform, Username: "platform-admin", Password: "password"}) {
		t.Fatal("PLATFORM login must be valid without appId or merchantId")
	}
}

func TestTokenAndAccountErrorHelpers(t *testing.T) {
	first, second := randomToken(32), randomToken(32)
	if first == second || len(first) != 43 || len(second) != 43 {
		t.Fatalf("refresh tokens are not unique 256-bit values: %q %q", first, second)
	}
	if !bytes.Equal(tokenHash(first), tokenHash(first)) || bytes.Equal(tokenHash(first), tokenHash(second)) {
		t.Fatal("token hash is not deterministic or collision-resistant for the fixture")
	}
	for _, code := range []string{"23505", "40001"} {
		if !errors.Is(accountWriteError(&pgconn.PgError{Code: code}), ErrAccountConflict) {
			t.Fatalf("PostgreSQL conflict %s was not mapped", code)
		}
	}
	sentinel := errors.New("database unavailable")
	if !errors.Is(accountWriteError(sentinel), sentinel) {
		t.Fatal("unrelated database error was changed")
	}
	if _, err := New(nil, nil); err == nil {
		t.Fatal("nil identity dependencies were accepted")
	}
}

func TestMerchantLoginStillRequiresTenant(t *testing.T) {
	if validLogin(LoginInput{Realm: accessidentity.RealmMerchant, Username: "merchant-admin", Password: "password"}) {
		t.Fatal("MERCHANT login without appId and merchantId must be rejected")
	}
	if !validLogin(LoginInput{Realm: accessidentity.RealmMerchant, AppID: 1001, MerchantID: 2001, Username: "merchant-admin", Password: "password"}) {
		t.Fatal("MERCHANT login with a valid tenant must be accepted")
	}
}
