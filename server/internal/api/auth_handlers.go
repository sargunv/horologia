package api

import (
	"context"
	"time"

	apigen "github.com/sargunv/tend/server/internal/api/gen"
	"github.com/sargunv/tend/server/internal/auth"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

// --- Auth: Tokens ---

func (h *Handler) AuthListTokens(ctx context.Context) (*apigen.AuthTokenList, error) {
	user := auth.UserFromContext(ctx)
	now := time.Now()

	q := dbgen.New(h.Pool)
	tokens, err := q.ListAuthTokensByUser(ctx, dbgen.ListAuthTokensByUserParams{
		UserID:    user.ID,
		ExpiresAt: types.Timestamptz(now),
	})
	if err != nil {
		return nil, err
	}

	items, err := convertEach(authTokenFromDB)(tokens)
	if err != nil {
		return nil, err
	}

	return &apigen.AuthTokenList{Items: items}, nil
}

func (h *Handler) AuthCreateToken(ctx context.Context, req *apigen.AuthTokenCreate) (*apigen.AuthTokenCreateResponse, error) {
	user := auth.UserFromContext(ctx)

	raw, hash, err := generateToken()
	if err != nil {
		return nil, err
	}

	q := dbgen.New(h.Pool)
	token, err := q.CreateAuthToken(ctx, dbgen.CreateAuthTokenParams{
		UserID:    user.ID,
		TokenHash: hash,
		Name:      req.Name,
		Kind:      dbgen.AuthTokenKindApi,
		CreatedAt: types.Timestamptz(time.Now()),
	})
	if err != nil {
		return nil, err
	}

	apiToken := authTokenFromDB(token)

	return &apigen.AuthTokenCreateResponse{
		Token:     raw,
		AuthToken: *apiToken,
	}, nil
}

func (h *Handler) AuthDeleteToken(ctx context.Context, params apigen.AuthDeleteTokenParams) error {
	user := auth.UserFromContext(ctx)

	id, err := parseTokenID(params.TokenId)
	if err != nil {
		return badRequest(err.Error())
	}

	q := dbgen.New(h.Pool)
	result, err := q.DeleteAuthToken(ctx, dbgen.DeleteAuthTokenParams{
		ID:     id,
		UserID: user.ID,
	})
	if err != nil {
		return err
	}
	return checkDeleted(result)
}
