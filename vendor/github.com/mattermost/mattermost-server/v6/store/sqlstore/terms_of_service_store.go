// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package sqlstore

import (
	"database/sql"

	"github.com/mattermost/mattermost-server/v6/einterfaces"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/store"
	"github.com/pkg/errors"
)

type SqlTermsOfServiceStore struct {
	*SqlStore
	metrics einterfaces.MetricsInterface
}

func newSqlTermsOfServiceStore(sqlStore *SqlStore, metrics einterfaces.MetricsInterface) store.TermsOfServiceStore {
	s := SqlTermsOfServiceStore{sqlStore, metrics}

	for _, db := range sqlStore.GetAllConns() {
		table := db.AddTableWithName(model.TermsOfService{}, "TermsOfService").SetKeys(false, "Id")
		table.ColMap("Id").SetMaxSize(26)
		table.ColMap("UserId").SetMaxSize(26)
		table.ColMap("Text").SetMaxSize(model.PostMessageMaxBytesV2)
	}

	return s
}

func (s SqlTermsOfServiceStore) createIndexesIfNotExists() {
}

func (s SqlTermsOfServiceStore) Save(termsOfService *model.TermsOfService) (*model.TermsOfService, error) {
	if termsOfService.Id != "" {
		return nil, store.NewErrInvalidInput("TermsOfService", "Id", termsOfService.Id)
	}

	termsOfService.PreSave()

	if err := termsOfService.IsValid(); err != nil {
		return nil, err
	}
	query := `INSERT INTO TermsOfService
				(Id, CreateAt, UserId, Text)
				VALUES
				(:Id, :CreateAt, :UserId, :Text)
				`

	if _, err := s.GetMasterX().NamedExec(query, termsOfService); err != nil {
		return nil, errors.Wrapf(err, "could not save a new TermsOfService")
	}

	return termsOfService, nil
}

func (s SqlTermsOfServiceStore) GetLatest(allowFromCache bool) (*model.TermsOfService, error) {
	var termsOfService model.TermsOfService

	query := s.getQueryBuilder().
		Select("*").
		From("TermsOfService").
		OrderBy("CreateAt DESC").
		Limit(uint64(1))

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "could not build sql query to get latest TOS")
	}

	if err := s.GetReplicaX().Get(&termsOfService, queryString, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, store.NewErrNotFound("TermsOfService", "CreateAt=latest")
		}
		return nil, errors.Wrap(err, "could not find latest TermsOfService")
	}

	return &termsOfService, nil
}

func (s SqlTermsOfServiceStore) Get(id string, allowFromCache bool) (*model.TermsOfService, error) {
	var termsOfService model.TermsOfService
	queryString, _, err := s.getQueryBuilder().
		Select("*").
		From("TermsOfService").
		Where("id = ?").
		ToSql()

	if err != nil {
		return nil, errors.Wrap(err, "terms_of_service_to_sql")
	}

	err = s.GetReplicaX().Get(&termsOfService, queryString, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, store.NewErrNotFound("TermsOfService", "id")
		}
		return nil, errors.Wrapf(err, "could not find TermsOfService with id=%s", id)
	}
	return &termsOfService, nil
}
