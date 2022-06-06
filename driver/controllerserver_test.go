package driver

import (
	"reflect"
	"testing"
)

func Test_parseTags(t *testing.T) {
	type args struct {
		p string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]string
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			args: args{
				p: "dfa~daffd;fdasf~d12223=;ff~;hh",
			},
			want: map[string]string{
				"dfa":   "daffd",
				"fdasf": "d12223",
				// "":"",
				// "":"",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTags(tt.args.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseTags() = %v, want %v", got, tt.want)
			}
		})
	}
}
