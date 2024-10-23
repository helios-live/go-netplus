package netplus

import (
	"net"
	"reflect"
	"testing"
	"time"
)

type MockListener struct{}

func (ml *MockListener) Accept() (net.Conn, error) {
	return nil, nil
}

func (ml *MockListener) Close() error {
	return nil
}

func (ml *MockListener) Addr() net.Addr {
	return nil
}

func TestCounterListener_Accept(t *testing.T) {
	type fields struct {
		Listener   net.Listener
		rpm        [MaxMinutes]int64
		lastMinute int64
	}
	tests := []struct {
		name    string
		fields  *fields
		rounds  int
		want    net.Conn
		wantErr bool
	}{
		{
			name: "Basic Accept",
			fields: &fields{
				Listener:   &MockListener{},
				rpm:        [MaxMinutes]int64{},
				lastMinute: 0,
			},
			rounds:  1,
			want:    &CounterConn{},
			wantErr: false,
		},
		{
			name: "Basic Accept",
			fields: &fields{
				Listener:   &MockListener{},
				rpm:        [MaxMinutes]int64{1581},
				lastMinute: 0,
			},
			rounds:  815,
			want:    &CounterConn{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := NewCounterListener(&MockListener{})
			for i := 0; i < tt.rounds; i++ {
				// Simulate a minute passing
				if i%60+1 == 1 && i != 0 {
					now := time.Now().Unix()
					cl.LastMinute = &now
				}
				got, err := cl.Accept()
				if (err != nil) != tt.wantErr {
					t.Errorf("CounterListener.Accept() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("CounterListener.Accept() = %v, want %v", got, tt.want)
				}
				if cl.GetRPM()[0] != int64(i%60+1) {
					t.Errorf("CounterListener.Accept() = %v, want %v", cl.GetRPM()[0], i%60+1)
				}
			}
		})
	}
}
