package utils

import "testing"

func TestBFSDirectoryTraversal(t *testing.T) {
	type args struct {
		root     string
		maxItems int
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "",
			args: args{
				root:     "../../",
				maxItems: 200,
			},
			want:    "",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BFSDirectoryTraversal(tt.args.root, tt.args.maxItems)
			if (err != nil) != tt.wantErr {
				t.Errorf("BFSDirectoryTraversal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("BFSDirectoryTraversal() = %v, want %v", got, tt.want)
			}
		})
	}
}
