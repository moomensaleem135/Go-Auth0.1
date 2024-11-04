// +build go1.7

// Package conformance provides conformance tests for storage implementations.
package conformance

import (
	"reflect"
	"sort"
	"testing"
	"time"

	jose "gopkg.in/square/go-jose.v2"

	"golang.org/x/crypto/bcrypt"

	"github.com/coreos/dex/storage"

	"github.com/kylelemons/godebug/pretty"
)

// ensure that values being tested on never expire.
var neverExpire = time.Now().UTC().Add(time.Hour * 24 * 365 * 100)

type subTest struct {
	name string
	run  func(t *testing.T, s storage.Storage)
}

func runTests(t *testing.T, newStorage func() storage.Storage, tests []subTest) {
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := newStorage()
			test.run(t, s)
			s.Close()
		})
	}
}

// RunTests runs a set of conformance tests against a storage. newStorage should
// return an initialized but empty storage. The storage will be closed at the
// end of each test run.
func RunTests(t *testing.T, newStorage func() storage.Storage) {
	runTests(t, newStorage, []subTest{
		{"AuthCodeCRUD", testAuthCodeCRUD},
		{"AuthRequestCRUD", testAuthRequestCRUD},
		{"ClientCRUD", testClientCRUD},
		{"RefreshTokenCRUD", testRefreshTokenCRUD},
		{"PasswordCRUD", testPasswordCRUD},
		{"KeysCRUD", testKeysCRUD},
		{"OfflineSessionCRUD", testOfflineSessionCRUD},
		{"GarbageCollection", testGC},
		{"TimezoneSupport", testTimezones},
	})
}

func mustLoadJWK(b string) *jose.JSONWebKey {
	var jwt jose.JSONWebKey
	if err := jwt.UnmarshalJSON([]byte(b)); err != nil {
		panic(err)
	}
	return &jwt
}

func mustBeErrNotFound(t *testing.T, kind string, err error) {
	switch {
	case err == nil:
		t.Errorf("deleting non-existent %s should return an error", kind)
	case err != storage.ErrNotFound:
		t.Errorf("deleting %s expected storage.ErrNotFound, got %v", kind, err)
	}
}

func mustBeErrAlreadyExists(t *testing.T, kind string, err error) {
	switch {
	case err == nil:
		t.Errorf("attempting to create an existing %s should return an error", kind)
	case err != storage.ErrAlreadyExists:
		t.Errorf("creating an existing %s expected storage.ErrAlreadyExists, got %v", kind, err)
	}
}

func testAuthRequestCRUD(t *testing.T, s storage.Storage) {
	a := storage.AuthRequest{
		ID:                  storage.NewID(),
		ClientID:            "foobar",
		ResponseTypes:       []string{"code"},
		Scopes:              []string{"openid", "email"},
		RedirectURI:         "https://localhost:80/callback",
		Nonce:               "foo",
		State:               "bar",
		ForceApprovalPrompt: true,
		LoggedIn:            true,
		Expiry:              neverExpire,
		ConnectorID:         "ldap",
		ConnectorData:       []byte(`{"some":"data"}`),
		Claims: storage.Claims{
			UserID:        "1",
			Username:      "jane",
			Email:         "jane.doe@example.com",
			EmailVerified: true,
			Groups:        []string{"a", "b"},
		},
	}

	identity := storage.Claims{Email: "foobar"}

	if err := s.CreateAuthRequest(a); err != nil {
		t.Fatalf("failed creating auth request: %v", err)
	}

	// Attempt to create same AuthRequest twice.
	err := s.CreateAuthRequest(a)
	mustBeErrAlreadyExists(t, "auth request", err)

	if err := s.UpdateAuthRequest(a.ID, func(old storage.AuthRequest) (storage.AuthRequest, error) {
		old.Claims = identity
		old.ConnectorID = "connID"
		return old, nil
	}); err != nil {
		t.Fatalf("failed to update auth request: %v", err)
	}

	got, err := s.GetAuthRequest(a.ID)
	if err != nil {
		t.Fatalf("failed to get auth req: %v", err)
	}
	if !reflect.DeepEqual(got.Claims, identity) {
		t.Fatalf("update failed, wanted identity=%#v got %#v", identity, got.Claims)
	}
}

func testAuthCodeCRUD(t *testing.T, s storage.Storage) {
	a := storage.AuthCode{
		ID:            storage.NewID(),
		ClientID:      "foobar",
		RedirectURI:   "https://localhost:80/callback",
		Nonce:         "foobar",
		Scopes:        []string{"openid", "email"},
		Expiry:        neverExpire,
		ConnectorID:   "ldap",
		ConnectorData: []byte(`{"some":"data"}`),
		Claims: storage.Claims{
			UserID:        "1",
			Username:      "jane",
			Email:         "jane.doe@example.com",
			EmailVerified: true,
			Groups:        []string{"a", "b"},
		},
	}

	if err := s.CreateAuthCode(a); err != nil {
		t.Fatalf("failed creating auth code: %v", err)
	}

	// Attempt to create same AuthCode twice.
	err := s.CreateAuthCode(a)
	mustBeErrAlreadyExists(t, "auth code", err)

	got, err := s.GetAuthCode(a.ID)
	if err != nil {
		t.Fatalf("failed to get auth req: %v", err)
	}
	if a.Expiry.Unix() != got.Expiry.Unix() {
		t.Errorf("auth code expiry did not match want=%s vs got=%s", a.Expiry, got.Expiry)
	}
	got.Expiry = a.Expiry // time fields do not compare well
	if diff := pretty.Compare(a, got); diff != "" {
		t.Errorf("auth code retrieved from storage did not match: %s", diff)
	}

	if err := s.DeleteAuthCode(a.ID); err != nil {
		t.Fatalf("delete auth code: %v", err)
	}

	_, err = s.GetAuthCode(a.ID)
	mustBeErrNotFound(t, "auth code", err)
}

func testClientCRUD(t *testing.T, s storage.Storage) {
	id := storage.NewID()
	c := storage.Client{
		ID:           id,
		Secret:       "foobar",
		RedirectURIs: []string{"foo://bar.com/", "https://auth.example.com"},
		Name:         "dex client",
		LogoURL:      "https://goo.gl/JIyzIC",
	}
	err := s.DeleteClient(id)
	mustBeErrNotFound(t, "client", err)

	if err := s.CreateClient(c); err != nil {
		t.Fatalf("create client: %v", err)
	}

	// Attempt to create same Client twice.
	err = s.CreateClient(c)
	mustBeErrAlreadyExists(t, "client", err)

	getAndCompare := func(id string, want storage.Client) {
		gc, err := s.GetClient(id)
		if err != nil {
			t.Errorf("get client: %v", err)
			return
		}
		if diff := pretty.Compare(want, gc); diff != "" {
			t.Errorf("client retrieved from storage did not match: %s", diff)
		}
	}

	getAndCompare(id, c)

	newSecret := "barfoo"
	err = s.UpdateClient(id, func(old storage.Client) (storage.Client, error) {
		old.Secret = newSecret
		return old, nil
	})
	if err != nil {
		t.Errorf("update client: %v", err)
	}
	c.Secret = newSecret
	getAndCompare(id, c)

	if err := s.DeleteClient(id); err != nil {
		t.Fatalf("delete client: %v", err)
	}

	_, err = s.GetClient(id)
	mustBeErrNotFound(t, "client", err)
}

func testRefreshTokenCRUD(t *testing.T, s storage.Storage) {
	id := storage.NewID()
	refresh := storage.RefreshToken{
		ID:          id,
		Token:       "bar",
		Nonce:       "foo",
		ClientID:    "client_id",
		ConnectorID: "client_secret",
		Scopes:      []string{"openid", "email", "profile"},
		CreatedAt:   time.Now().UTC().Round(time.Millisecond),
		LastUsed:    time.Now().UTC().Round(time.Millisecond),
		Claims: storage.Claims{
			UserID:        "1",
			Username:      "jane",
			Email:         "jane.doe@example.com",
			EmailVerified: true,
			Groups:        []string{"a", "b"},
		},
		ConnectorData: []byte(`{"some":"data"}`),
	}
	if err := s.CreateRefresh(refresh); err != nil {
		t.Fatalf("create refresh token: %v", err)
	}

	// Attempt to create same Refresh Token twice.
	err := s.CreateRefresh(refresh)
	mustBeErrAlreadyExists(t, "refresh token", err)

	getAndCompare := func(id string, want storage.RefreshToken) {
		gr, err := s.GetRefresh(id)
		if err != nil {
			t.Errorf("get refresh: %v", err)
			return
		}
		if diff := pretty.Compare(want, gr); diff != "" {
			t.Errorf("refresh token retrieved from storage did not match: %s", diff)
		}
	}

	getAndCompare(id, refresh)

	id2 := storage.NewID()
	refresh2 := storage.RefreshToken{
		ID:          id2,
		Token:       "bar_2",
		Nonce:       "foo_2",
		ClientID:    "client_id_2",
		ConnectorID: "client_secret",
		Scopes:      []string{"openid", "email", "profile"},
		CreatedAt:   time.Now().UTC().Round(time.Millisecond),
		LastUsed:    time.Now().UTC().Round(time.Millisecond),
		Claims: storage.Claims{
			UserID:        "2",
			Username:      "john",
			Email:         "john.doe@example.com",
			EmailVerified: true,
			Groups:        []string{"a", "b"},
		},
		ConnectorData: []byte(`{"some":"data"}`),
	}

	if err := s.CreateRefresh(refresh2); err != nil {
		t.Fatalf("create second refresh token: %v", err)
	}

	getAndCompare(id2, refresh2)

	updatedAt := time.Now().UTC().Round(time.Millisecond)

	updater := func(r storage.RefreshToken) (storage.RefreshToken, error) {
		r.Token = "spam"
		r.LastUsed = updatedAt
		return r, nil
	}
	if err := s.UpdateRefreshToken(id, updater); err != nil {
		t.Errorf("failed to udpate refresh token: %v", err)
	}
	refresh.Token = "spam"
	refresh.LastUsed = updatedAt
	getAndCompare(id, refresh)

	// Ensure that updating the first token doesn't impact the second. Issue #847.
	getAndCompare(id2, refresh2)

	if err := s.DeleteRefresh(id); err != nil {
		t.Fatalf("failed to delete refresh request: %v", err)
	}

	_, err = s.GetRefresh(id)
	mustBeErrNotFound(t, "refresh token", err)
}

type byEmail []storage.Password

func (n byEmail) Len() int           { return len(n) }
func (n byEmail) Less(i, j int) bool { return n[i].Email < n[j].Email }
func (n byEmail) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }

func testPasswordCRUD(t *testing.T, s storage.Storage) {
	// Use bcrypt.MinCost to keep the tests short.
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}

	password := storage.Password{
		Email:    "jane@example.com",
		Hash:     passwordHash,
		Username: "jane",
		UserID:   "foobar",
	}
	if err := s.CreatePassword(password); err != nil {
		t.Fatalf("create password token: %v", err)
	}

	// Attempt to create same Password twice.
	err = s.CreatePassword(password)
	mustBeErrAlreadyExists(t, "password", err)

	getAndCompare := func(id string, want storage.Password) {
		gr, err := s.GetPassword(id)
		if err != nil {
			t.Errorf("get password %q: %v", id, err)
			return
		}
		if diff := pretty.Compare(want, gr); diff != "" {
			t.Errorf("password retrieved from storage did not match: %s", diff)
		}
	}

	getAndCompare("jane@example.com", password)
	getAndCompare("JANE@example.com", password) // Emails should be case insensitive

	if err := s.UpdatePassword(password.Email, func(old storage.Password) (storage.Password, error) {
		old.Username = "jane doe"
		return old, nil
	}); err != nil {
		t.Fatalf("failed to update auth request: %v", err)
	}

	password.Username = "jane doe"
	getAndCompare("jane@example.com", password)

	var passwordList []storage.Password
	passwordList = append(passwordList, password)

	listAndCompare := func(want []storage.Password) {
		passwords, err := s.ListPasswords()
		if err != nil {
			t.Errorf("list password: %v", err)
			return
		}
		sort.Sort(byEmail(want))
		sort.Sort(byEmail(passwords))
		if diff := pretty.Compare(want, passwords); diff != "" {
			t.Errorf("password list retrieved from storage did not match: %s", diff)
		}
	}

	listAndCompare(passwordList)

	if err := s.DeletePassword(password.Email); err != nil {
		t.Fatalf("failed to delete password: %v", err)
	}

	_, err = s.GetPassword(password.Email)
	mustBeErrNotFound(t, "password", err)

}

func testOfflineSessionCRUD(t *testing.T, s storage.Storage) {
	session := storage.OfflineSessions{
		UserID:  "User",
		ConnID:  "Conn",
		Refresh: make(map[string]*storage.RefreshTokenRef),
	}

	// Creating an OfflineSession with an empty Refresh list to ensure that
	// an empty map is translated as expected by the storage.
	if err := s.CreateOfflineSessions(session); err != nil {
		t.Fatalf("create offline session: %v", err)
	}

	// Attempt to create same OfflineSession twice.
	err := s.CreateOfflineSessions(session)
	mustBeErrAlreadyExists(t, "offline session", err)

	getAndCompare := func(userID string, connID string, want storage.OfflineSessions) {
		gr, err := s.GetOfflineSessions(userID, connID)
		if err != nil {
			t.Errorf("get offline session: %v", err)
			return
		}
		if diff := pretty.Compare(want, gr); diff != "" {
			t.Errorf("offline session retrieved from storage did not match: %s", diff)
		}
	}

	getAndCompare("User", "Conn", session)

	id := storage.NewID()
	tokenRef := storage.RefreshTokenRef{
		ID:        id,
		ClientID:  "client_id",
		CreatedAt: time.Now().UTC().Round(time.Millisecond),
		LastUsed:  time.Now().UTC().Round(time.Millisecond),
	}
	session.Refresh[tokenRef.ClientID] = &tokenRef

	if err := s.UpdateOfflineSessions(session.UserID, session.ConnID, func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
		old.Refresh[tokenRef.ClientID] = &tokenRef
		return old, nil
	}); err != nil {
		t.Fatalf("failed to update offline session: %v", err)
	}

	getAndCompare("User", "Conn", session)

	if err := s.DeleteOfflineSessions(session.UserID, session.ConnID); err != nil {
		t.Fatalf("failed to delete offline session: %v", err)
	}

	_, err = s.GetOfflineSessions(session.UserID, session.ConnID)
	mustBeErrNotFound(t, "offline session", err)

}

func testKeysCRUD(t *testing.T, s storage.Storage) {
	updateAndCompare := func(k storage.Keys) {
		err := s.UpdateKeys(func(oldKeys storage.Keys) (storage.Keys, error) {
			return k, nil
		})
		if err != nil {
			t.Errorf("failed to update keys: %v", err)
			return
		}

		if got, err := s.GetKeys(); err != nil {
			t.Errorf("failed to get keys: %v", err)
		} else {
			got.NextRotation = got.NextRotation.UTC()
			if diff := pretty.Compare(k, got); diff != "" {
				t.Errorf("got keys did not equal expected: %s", diff)
			}
		}
	}

	// Postgres isn't as accurate with nano seconds as we'd like
	n := time.Now().UTC().Round(time.Second)

	keys1 := storage.Keys{
		SigningKey:    jsonWebKeys[0].Private,
		SigningKeyPub: jsonWebKeys[0].Public,
		NextRotation:  n,
	}

	keys2 := storage.Keys{
		SigningKey:    jsonWebKeys[2].Private,
		SigningKeyPub: jsonWebKeys[2].Public,
		NextRotation:  n.Add(time.Hour),
		VerificationKeys: []storage.VerificationKey{
			{
				PublicKey: jsonWebKeys[0].Public,
				Expiry:    n.Add(time.Hour),
			},
			{
				PublicKey: jsonWebKeys[1].Public,
				Expiry:    n.Add(time.Hour * 2),
			},
		},
	}

	updateAndCompare(keys1)
	updateAndCompare(keys2)
}

func testGC(t *testing.T, s storage.Storage) {
	est, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	pst, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}

	expiry := time.Now().In(est)
	c := storage.AuthCode{
		ID:            storage.NewID(),
		ClientID:      "foobar",
		RedirectURI:   "https://localhost:80/callback",
		Nonce:         "foobar",
		Scopes:        []string{"openid", "email"},
		Expiry:        expiry,
		ConnectorID:   "ldap",
		ConnectorData: []byte(`{"some":"data"}`),
		Claims: storage.Claims{
			UserID:        "1",
			Username:      "jane",
			Email:         "jane.doe@example.com",
			EmailVerified: true,
			Groups:        []string{"a", "b"},
		},
	}

	if err := s.CreateAuthCode(c); err != nil {
		t.Fatalf("failed creating auth code: %v", err)
	}

	for _, tz := range []*time.Location{time.UTC, est, pst} {
		result, err := s.GarbageCollect(expiry.Add(-time.Hour).In(tz))
		if err != nil {
			t.Errorf("garbage collection failed: %v", err)
		} else {
			if result.AuthCodes != 0 || result.AuthRequests != 0 {
				t.Errorf("expected no garbage collection results, got %#v", result)
			}
		}
		if _, err := s.GetAuthCode(c.ID); err != nil {
			t.Errorf("expected to be able to get auth code after GC: %v", err)
		}
	}

	if r, err := s.GarbageCollect(expiry.Add(time.Hour)); err != nil {
		t.Errorf("garbage collection failed: %v", err)
	} else if r.AuthCodes != 1 {
		t.Errorf("expected to garbage collect 1 objects, got %d", r.AuthCodes)
	}

	if _, err := s.GetAuthCode(c.ID); err == nil {
		t.Errorf("expected auth code to be GC'd")
	} else if err != storage.ErrNotFound {
		t.Errorf("expected storage.ErrNotFound, got %v", err)
	}

	a := storage.AuthRequest{
		ID:                  storage.NewID(),
		ClientID:            "foobar",
		ResponseTypes:       []string{"code"},
		Scopes:              []string{"openid", "email"},
		RedirectURI:         "https://localhost:80/callback",
		Nonce:               "foo",
		State:               "bar",
		ForceApprovalPrompt: true,
		LoggedIn:            true,
		Expiry:              expiry,
		ConnectorID:         "ldap",
		ConnectorData:       []byte(`{"some":"data"}`),
		Claims: storage.Claims{
			UserID:        "1",
			Username:      "jane",
			Email:         "jane.doe@example.com",
			EmailVerified: true,
			Groups:        []string{"a", "b"},
		},
	}

	if err := s.CreateAuthRequest(a); err != nil {
		t.Fatalf("failed creating auth request: %v", err)
	}

	for _, tz := range []*time.Location{time.UTC, est, pst} {
		result, err := s.GarbageCollect(expiry.Add(-time.Hour).In(tz))
		if err != nil {
			t.Errorf("garbage collection failed: %v", err)
		} else {
			if result.AuthCodes != 0 || result.AuthRequests != 0 {
				t.Errorf("expected no garbage collection results, got %#v", result)
			}
		}
		if _, err := s.GetAuthRequest(a.ID); err != nil {
			t.Errorf("expected to be able to get auth code after GC: %v", err)
		}
	}

	if r, err := s.GarbageCollect(expiry.Add(time.Hour)); err != nil {
		t.Errorf("garbage collection failed: %v", err)
	} else if r.AuthRequests != 1 {
		t.Errorf("expected to garbage collect 1 objects, got %d", r.AuthRequests)
	}

	if _, err := s.GetAuthRequest(a.ID); err == nil {
		t.Errorf("expected auth code to be GC'd")
	} else if err != storage.ErrNotFound {
		t.Errorf("expected storage.ErrNotFound, got %v", err)
	}
}

// testTimezones tests that backends either fully support timezones or
// do the correct standardization.
func testTimezones(t *testing.T, s storage.Storage) {
	est, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	// Create an expiry with timezone info. Only expect backends to be
	// accurate to the millisecond
	expiry := time.Now().In(est).Round(time.Millisecond)

	c := storage.AuthCode{
		ID:            storage.NewID(),
		ClientID:      "foobar",
		RedirectURI:   "https://localhost:80/callback",
		Nonce:         "foobar",
		Scopes:        []string{"openid", "email"},
		Expiry:        expiry,
		ConnectorID:   "ldap",
		ConnectorData: []byte(`{"some":"data"}`),
		Claims: storage.Claims{
			UserID:        "1",
			Username:      "jane",
			Email:         "jane.doe@example.com",
			EmailVerified: true,
			Groups:        []string{"a", "b"},
		},
	}
	if err := s.CreateAuthCode(c); err != nil {
		t.Fatalf("failed creating auth code: %v", err)
	}
	got, err := s.GetAuthCode(c.ID)
	if err != nil {
		t.Fatalf("failed to get auth code: %v", err)
	}

	// Ensure that if the resulting time is converted to the same
	// timezone, it's the same value. We DO NOT expect timezones
	// to be preserved.
	gotTime := got.Expiry.In(est)
	wantTime := expiry
	if !gotTime.Equal(wantTime) {
		t.Fatalf("expected expiry %v got %v", wantTime, gotTime)
	}
}
