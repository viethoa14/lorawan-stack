// Copyright © 2018 The Things Network Foundation, The Things Industries B.V.
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

package applicationserver

import (
	"context"
	"fmt"
	"time"

	"go.thethings.network/lorawan-stack/pkg/applicationserver/io"
	"go.thethings.network/lorawan-stack/pkg/component"
	"go.thethings.network/lorawan-stack/pkg/errors"
	"go.thethings.network/lorawan-stack/pkg/events"
	"go.thethings.network/lorawan-stack/pkg/log"
	"go.thethings.network/lorawan-stack/pkg/rpcclient"
	"go.thethings.network/lorawan-stack/pkg/rpcmetadata"
	"go.thethings.network/lorawan-stack/pkg/ttnpb"
	"go.thethings.network/lorawan-stack/pkg/unique"
	"google.golang.org/grpc"
)

func (as *ApplicationServer) linkAll(ctx context.Context) error {
	return as.linkRegistry.Range(
		ctx,
		[]string{
			"network_server_address",
			"api_key",
			"default_formatters",
		},
		func(ctx context.Context, ids ttnpb.ApplicationIdentifiers, target *ttnpb.ApplicationLink) bool {
			as.startLinkTask(ctx, ids, target)
			return true
		},
	)
}

var linkBackoff = []time.Duration{100 * time.Millisecond, 1 * time.Second, 10 * time.Second}

func (as *ApplicationServer) startLinkTask(ctx context.Context, ids ttnpb.ApplicationIdentifiers, target *ttnpb.ApplicationLink) {
	// TODO: Add jitter to the backoff (https://github.com/TheThingsIndustries/lorawan-stack/issues/1227)
	as.StartTask(ctx, func(ctx context.Context) error {
		err := as.link(ctx, ids, target)
		switch {
		case errors.IsFailedPrecondition(err), errors.IsUnauthenticated(err), errors.IsPermissionDenied(err):
			log.FromContext(ctx).WithError(err).Warn("Failed to link")
			return nil
		case errors.IsCanceled(err), errors.IsAlreadyExists(err):
			return nil
		default:
			return err
		}
	}, component.TaskRestartOnFailure, linkBackoff...)
}

type link struct {
	ttnpb.ApplicationLink
	ctx    context.Context
	cancel context.CancelFunc

	conn         *grpc.ClientConn
	connName     string
	connCallOpts []grpc.CallOption
	connReady    chan struct{}

	subscribeCh   chan *io.Connection
	unsubscribeCh chan *io.Connection
	upCh          chan *ttnpb.ApplicationUp
}

const linkBufferSize = 10

var (
	errAlreadyLinked = errors.DefineAlreadyExists("already_linked", "already linked to `{application_uid}`")
	errNSNotFound    = errors.DefineNotFound("network_server_not_found", "Network Server not found for `{application_uid}`")
)

func (as *ApplicationServer) connectLink(ctx context.Context, ids ttnpb.ApplicationIdentifiers, link *link) error {
	if link.NetworkServerAddress != "" {
		options := rpcclient.DefaultDialOptions(ctx)
		if link.AllowInsecure {
			options = append(options, grpc.WithInsecure())
		}
		conn, err := grpc.DialContext(ctx, link.NetworkServerAddress, options...)
		if err != nil {
			return err
		}
		link.conn = conn
		link.connName = link.NetworkServerAddress
		link.connCallOpts = []grpc.CallOption{
			grpc.PerRPCCredentials(rpcmetadata.MD{
				AuthType:      "Key",
				AuthValue:     link.APIKey,
				AllowInsecure: link.AllowInsecure,
			}),
		}
		go func() {
			<-ctx.Done()
			conn.Close()
		}()
	} else {
		ns := as.GetPeer(ctx, ttnpb.PeerInfo_NETWORK_SERVER, ids)
		if ns == nil {
			return errNSNotFound.WithAttributes("application_uid", unique.ID(ctx, ids))
		}
		link.conn = ns.Conn()
		link.connName = ns.Name()
		link.connCallOpts = []grpc.CallOption{
			as.WithClusterAuth(),
		}
	}
	close(link.connReady)
	return nil
}

func (as *ApplicationServer) link(ctx context.Context, ids ttnpb.ApplicationIdentifiers, target *ttnpb.ApplicationLink) error {
	uid := unique.ID(ctx, ids)
	ctx = log.NewContextWithField(ctx, "application_uid", uid)
	ctx, cancel := context.WithCancel(ctx)
	l := &link{
		ApplicationLink: *target,
		ctx:             ctx,
		cancel:          cancel,
		connReady:       make(chan struct{}),
		subscribeCh:     make(chan *io.Connection, 1),
		unsubscribeCh:   make(chan *io.Connection, 1),
		upCh:            make(chan *ttnpb.ApplicationUp, linkBufferSize),
	}
	if _, loaded := as.links.LoadOrStore(uid, l); loaded {
		cancel()
		return errAlreadyLinked.WithAttributes("application_uid", uid)
	}
	defer func() {
		cancel()
		as.links.Delete(uid)
	}()
	if err := as.connectLink(ctx, ids, l); err != nil {
		return err
	}
	client := ttnpb.NewAsNsClient(l.conn)
	logger := log.FromContext(ctx).WithField("network_server", l.connName)
	logger.Debug("Linking")
	stream, err := client.LinkApplication(ctx, &ids, l.connCallOpts...)
	if err != nil {
		logger.WithError(err).Warn("Linking failed")
		return err
	}
	logger.Info("Linked")

	go l.run()
	for _, sub := range as.defaultSubscribers {
		sub := sub
		l.subscribeCh <- sub
		go func() {
			<-sub.Context().Done()
			l.unsubscribeCh <- sub
		}()
	}
	for {
		up, err := stream.Recv()
		if err != nil {
			if errors.IsCanceled(err) {
				logger.Debug("Unlinked")
			} else {
				logger.WithError(err).Warn("Link failed")
			}
			return err
		}
		registerReceiveUp(ctx, up, l.connName)
		if err := as.handleUp(ctx, up, l); err != nil {
			logger.WithError(err).Warn("Failed to process upstream message")
			registerDropUp(ctx, up, err)
			continue
		}
		switch p := up.Up.(type) {
		case *ttnpb.ApplicationUp_JoinAccept:
			p.JoinAccept.AppSKey = nil
			p.JoinAccept.InvalidatedDownlinks = nil
		case *ttnpb.ApplicationUp_DownlinkQueueInvalidated:
			continue
		}
		l.upCh <- up
		registerForwardUp(ctx, up)
	}
}

var errNotLinked = errors.DefineNotFound("not_linked", "not linked to `{application_uid}`")

func (as *ApplicationServer) cancelLink(ctx context.Context, ids ttnpb.ApplicationIdentifiers) error {
	l, err := as.getLink(ctx, ids)
	if err != nil {
		return err
	}
	uid := unique.ID(ctx, ids)
	log.FromContext(ctx).WithField("application_uid", uid).Debug("Unlinking")
	l.cancel()
	as.links.Delete(uid)
	return nil
}

func (as *ApplicationServer) getLink(ctx context.Context, ids ttnpb.ApplicationIdentifiers) (*link, error) {
	uid := unique.ID(ctx, ids)
	val, ok := as.links.Load(uid)
	if !ok {
		return nil, errNotLinked.WithAttributes("application_uid", uid)
	}
	return val.(*link), nil
}

func (l *link) run() {
	subscribers := make(map[*io.Connection]string)
	for {
		select {
		case <-l.ctx.Done():
			return
		case conn := <-l.subscribeCh:
			correlationID := fmt.Sprintf("subscriber:%s", events.NewCorrelationID())
			subscribers[conn] = correlationID
			registerSubscribe(events.ContextWithCorrelationID(l.ctx, correlationID), conn)
			log.FromContext(conn.Context()).Debug("Subscribed")
		case conn := <-l.unsubscribeCh:
			if correlationID, ok := subscribers[conn]; ok {
				delete(subscribers, conn)
				registerUnsubscribe(events.ContextWithCorrelationID(l.ctx, correlationID), conn)
				log.FromContext(conn.Context()).Debug("Unsubscribed")
			}
		case up := <-l.upCh:
			for conn := range subscribers {
				if err := conn.SendUp(up); err != nil {
					log.FromContext(conn.Context()).WithError(err).Warn("Send upstream message failed")
				}
			}
		}
	}
}
