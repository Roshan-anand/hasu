package security

import "testing"

func TestPassword(t *testing.T) {
	var hassPass string
	var err error
	pass := "testpassword123"
	wrongPass := "testwrongpassword"

	t.Run("hash the password", func(t *testing.T) {
		hassPass, err = HashPassword(pass)
		if err != nil {
			t.Fatal("err hashing passowrd : ", err)
		}
	})

	t.Run("match password and hash", func(t *testing.T) {
		if !CheckPasswordHash(pass, hassPass) {
			t.Fatal("password didnt match but want to match")
		}
	})

	t.Run("match wrong password and hash", func(t *testing.T) {
		if CheckPasswordHash(wrongPass, hassPass) {
			t.Fatal("password matched but want not match")
		}
	})
}
