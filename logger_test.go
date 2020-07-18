package logger

import (
	"testing"
)

func TestInitLogger(t *testing.T) {
	type args struct {
		opt *Options
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test",
			args: args{
				opt: &Options{
					Level:      DebugLevel,
					Filename:   "./log.log",
					MaxSize:    10,
					MaxBackups: 10,
					MaxAge:     2,
					Compress:   false,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitLogger(tt.args.opt)
			//Debugf("logger %s", time.Now().Format("20060102"))
			Pure("11", 22, "33")
			Puref("%s:%d", "hhh", 333)
		})
	}
}
