package socketio

import "github.com/robertkonga/yekonga-server/plugins/uuid"

func newV4UUID() string {
	return uuid.Must(uuid.NewV4()).String()
}
