// Copyright © 2019 The Things Network Foundation, The Things Industries B.V.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package store

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/smartystreets/assertions"
	"github.com/smartystreets/assertions/should"
	"go.thethings.network/lorawan-stack/v3/pkg/errors"
	"go.thethings.network/lorawan-stack/v3/pkg/ttnpb"
	"go.thethings.network/lorawan-stack/v3/pkg/util/test"
)

func TestAPIKeyStore(t *testing.T) {
	ctx := test.Context()

	WithDB(t, func(t *testing.T, db *gorm.DB) {
		prepareTest(db,
			&APIKey{},
			&Account{}, &User{}, &Organization{},
			&Application{}, &Client{}, &Gateway{},
		)

		s := newStore(db)
		store := GetAPIKeyStore(db)

		s.createEntity(ctx, &User{Account: Account{UID: "test-user"}})
		userIDs := &ttnpb.UserIdentifiers{UserID: "test-user"}

		s.createEntity(ctx, &Organization{Account: Account{UID: "test-org"}})
		orgIDs := &ttnpb.OrganizationIdentifiers{OrganizationID: "test-org"}

		s.createEntity(ctx, &Application{ApplicationID: "test-app"})
		appIDs := &ttnpb.ApplicationIdentifiers{ApplicationID: "test-app"}

		s.createEntity(ctx, &Gateway{GatewayID: "test-gtw"})
		gtwIDs := &ttnpb.GatewayIdentifiers{GatewayID: "test-gtw"}

		expiryTime := time.Now().Add(24 * time.Hour)
		expiryTimeString := expiryTime.Format("2006-01-02")
		expiryTimeValue, _ := time.Parse("2006-01-02", expiryTimeString)

		for _, tt := range []struct {
			Name        string
			Identifiers ttnpb.Identifiers
			Rights      []ttnpb.Right
		}{
			{
				Name:        "Application",
				Identifiers: appIDs,
				Rights:      []ttnpb.Right{ttnpb.RIGHT_APPLICATION_ALL},
			},
			{
				Name:        "Gateway",
				Identifiers: gtwIDs,
				Rights:      []ttnpb.Right{ttnpb.RIGHT_GATEWAY_ALL},
			},
			{
				Name:        "Organization",
				Identifiers: orgIDs,
				Rights:      []ttnpb.Right{ttnpb.RIGHT_APPLICATION_ALL, ttnpb.RIGHT_GATEWAY_ALL},
			},
			{
				Name:        "User",
				Identifiers: userIDs,
				Rights:      []ttnpb.Right{ttnpb.RIGHT_APPLICATION_ALL, ttnpb.RIGHT_GATEWAY_ALL},
			},
		} {
			t.Run(tt.Name, func(t *testing.T) {
				a := assertions.New(t)

				key := &ttnpb.APIKey{
					ID:        strings.ToUpper(fmt.Sprintf("%sKEYID", tt.Name)),
					Key:       strings.ToUpper(fmt.Sprintf("%sKEY", tt.Name)),
					Name:      fmt.Sprintf("%s API key", tt.Name),
					Rights:    tt.Rights,
					ExpiresAt: &expiryTimeValue,
				}
				created, err := store.CreateAPIKey(ctx, tt.Identifiers, key)

				a.So(err, should.BeNil)
				if a.So(created, should.NotBeNil) {
					a.So(created.ID, should.Equal, key.ID)
					a.So(created.Key, should.Equal, key.Key)
					a.So(created.Name, should.Equal, key.Name)
					a.So(created.Rights, should.Resemble, key.Rights)
					a.So(created.CreatedAt, should.HappenAfter, time.Now().Add(-1*time.Hour))
					a.So(created.UpdatedAt, should.HappenAfter, time.Now().Add(-1*time.Hour))
					a.So(*created.ExpiresAt, should.Equal, expiryTimeValue)
				}

				keys, err := store.FindAPIKeys(ctx, tt.Identifiers)

				a.So(err, should.BeNil)
				if a.So(keys, should.HaveLength, 1) {
					a.So(keys[0], should.Resemble, created)
				}

				ids, got, err := store.GetAPIKey(ctx, key.ID)

				a.So(err, should.BeNil)
				a.So(ids, should.Resemble, tt.Identifiers)
				a.So(got, should.Resemble, created)

				updated, err := store.UpdateAPIKey(ctx, tt.Identifiers, &ttnpb.APIKey{
					ID:     strings.ToUpper(fmt.Sprintf("%sKEYID", tt.Name)),
					Name:   fmt.Sprintf("Updated %s API key", tt.Name),
					Rights: tt.Rights,
				})

				a.So(err, should.BeNil)

				ids, got, err = store.GetAPIKey(ctx, key.ID)

				a.So(err, should.BeNil)
				a.So(ids, should.Resemble, tt.Identifiers)
				if a.So(got, should.NotBeNil) {
					a.So(got, should.Resemble, updated)
					a.So(got.Name, should.NotEqual, key.Name)
					a.So(got.Rights, should.Resemble, key.Rights)
				}

				updated, err = store.UpdateAPIKey(ctx, tt.Identifiers, &ttnpb.APIKey{
					ID: strings.ToUpper(fmt.Sprintf("%sKEYID", tt.Name)),
					// Empty rights
				})

				a.So(err, should.BeNil)
				a.So(updated, should.BeNil)

				_, _, err = store.GetAPIKey(ctx, key.ID)

				if a.So(err, should.NotBeNil) {
					a.So(errors.IsNotFound(err), should.BeTrue)
				}

				keys, err = store.FindAPIKeys(ctx, tt.Identifiers)

				a.So(err, should.BeNil)
				a.So(keys, should.HaveLength, 0)
			})
		}

		t.Run("Delete entity API keys", func(t *testing.T) {
			a := assertions.New(t)

			// Create three API keys for entity
			for i := 0; i < 3; i++ {
				key := &ttnpb.APIKey{
					ID:     strings.ToUpper(fmt.Sprintf("KEYID%d", i)),
					Key:    strings.ToUpper(fmt.Sprintf("KEY%d", i)),
					Name:   fmt.Sprintf("API key %d", i),
					Rights: []ttnpb.Right{ttnpb.RIGHT_APPLICATION_ALL},
				}

				store.CreateAPIKey(ctx, userIDs, key)
			}
			err := store.DeleteEntityAPIKeys(ctx, userIDs)

			a.So(err, should.BeNil)

			keys, err := store.FindAPIKeys(ctx, userIDs)

			// Check if all API keys are deleted
			a.So(err, should.BeNil)
			a.So(keys, should.HaveLength, 0)
		})
	})
}
