package oceanwallet

import (
	"context"
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"
	pb "github.com/vulpemventures/taxi-daemon/api-spec/protobuf/gen/go/ocean/v1"
	"github.com/vulpemventures/taxi-daemon/internal/core/ports"
	"google.golang.org/grpc"
)

type notifyManager struct {
	client  pb.NotificationServiceClient
	chUtxos chan ports.UtxoNotification
}

func newNotifyManager(conn *grpc.ClientConn) (ports.NotificationManager, error) {
	nm := &notifyManager{
		client:  pb.NewNotificationServiceClient(conn),
		chUtxos: make(chan ports.UtxoNotification),
	}
	chUtxoNotifications, err := nm.client.UtxosNotifications(
		context.Background(), &pb.UtxosNotificationsRequest{},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"error while opening stream to listen for utxo notifications: %s", err,
		)
	}
	go nm.startListeningToUtxoNotifications(chUtxoNotifications)
	return nm, nil
}

func (nm *notifyManager) UtxoChannel() chan ports.UtxoNotification {
	return nm.chUtxos
}

func (nm *notifyManager) startListeningToUtxoNotifications(
	stream pb.NotificationService_UtxosNotificationsClient,
) (err error) {
	defer func() {
		if err != nil {
			log.WithError(err).Fatal(
				"notification handler: error while listenting to utxo notifications",
			)
		}
	}()
	for {
		notification, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		// non-blocking write to channel
		select {
		case nm.chUtxos <- utxoNotification{notification}:
		default:
		}
	}
	return nil
}

type utxoNotification struct {
	*pb.UtxosNotificationsResponse
}

func (un utxoNotification) EventType() ports.UtxoEventType {
	return utxoEventType(un.GetEventType())
}
func (un utxoNotification) Utxos() []ports.Utxo {
	list := un.GetUtxos()
	utxos := make([]ports.Utxo, 0, len(list))
	for _, u := range list {
		utxos = append(utxos, utxo{u})
	}
	return utxos
}

type utxoEventType pb.UtxoEventType

func (ut utxoEventType) IsUnknown() bool {
	return int(ut) == int(pb.UtxoEventType_UTXO_EVENT_TYPE_UNSPECIFIED)
}
func (ut utxoEventType) IsNew() bool {
	return int(ut) == int(pb.UtxoEventType_UTXO_EVENT_TYPE_NEW)
}
func (ut utxoEventType) IsConfirmed() bool {
	return int(ut) == int(pb.UtxoEventType_UTXO_EVENT_TYPE_CONFIRMED)
}
func (ut utxoEventType) IsLocked() bool {
	return int(ut) == int(pb.UtxoEventType_UTXO_EVENT_TYPE_LOCKED)
}
func (ut utxoEventType) IsUnlocked() bool {
	return int(ut) == int(pb.UtxoEventType_UTXO_EVENT_TYPE_UNLOCKED)
}
func (ut utxoEventType) IsSpent() bool {
	return int(ut) == int(pb.UtxoEventType_UTXO_EVENT_TYPE_SPENT)
}

type utxo struct {
	*pb.Utxo
}

func (ui utxo) TxID() string {
	return ui.GetTxid()
}
func (ui utxo) Index() uint32 {
	return ui.GetIndex()
}
func (ui utxo) Asset() string {
	return ui.GetAsset()
}
func (ui utxo) Value() uint64 {
	return ui.GetValue()
}
func (ui utxo) Script() string {
	return ui.GetScript()
}
func (ui utxo) AssetBlinder() string {
	return ui.GetAssetBlinder()
}
func (ui utxo) ValueBlinder() string {
	return ui.GetValueBlinder()
}
func (ui utxo) AccountName() string {
	return ui.GetAccountName()
}
