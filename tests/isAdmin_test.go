package tests

import (
	"testing"

	ssov1 "github.com/AlexKorshun/protos/gen/go/sso"
	"github.com/AlexKorshun/sso/tests/suite"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsAdmib_NotFound(t *testing.T) {
	ctx, st := suite.New(t)

	userID := gofakeit.Int64()

	respIsAdmin, err := st.AuthClient.IsAdmin(ctx, &ssov1.IsAdminRequest{
		UserId: userID,
	})
	require.Error(t, err)
	assert.False(t, respIsAdmin.GetIsAdmin())

	assert.ErrorContains(t, err, "user not found")

}
