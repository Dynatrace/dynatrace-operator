package mutation

import (
	"testing"
)

func TestFeature_name(t *testing.T) {
	type fields struct {
		ftype   FeatureType
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "",
			fields: fields{
				ftype:   OneAgent,
			},
			want: "oneagent",
		},
		{
			name: "",
			fields: fields{
				ftype:   DataIngest,
			},
			want: "data-ingest",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Feature{
				ftype:   tt.fields.ftype,
			}
			if got := f.name(); got != tt.want {
				t.Errorf("name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInjectionInfo_enabled(t *testing.T) {
	features := func() map[Feature]struct{} {
		i := NewInjectionInfo()
		i.add(Feature{
			ftype:   OneAgent,
		})
		i.add(Feature{
			ftype:   DataIngest,
		})
		return i.features
	}()

	type args struct {
		wanted FeatureType
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "",
			args: args{
				wanted: OneAgent,
			},
			want: true,
		},
		{
			name: "",
			args: args{
				wanted: DataIngest,
			},
			want: true,
		},
		{
			name: "",
			args: args{
				wanted: 999,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &InjectionInfo{
				features: features,
			}
			if got := info.enabled(tt.args.wanted); got != tt.want {
				t.Errorf("enabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInjectionInfo_injectedAnnotation(t *testing.T) {
	type fields struct {
		features map[Feature]struct{}
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "OA-enabled",
			fields: fields{
				features: func() map[Feature]struct{} {
					i := NewInjectionInfo()
					i.add(Feature{
						ftype:   OneAgent,
					})
					return i.features
				}(),
			},
			want: "oneagent",
		},
		{
			name: "OA and DI enabled",
			fields: fields{
				features: func() map[Feature]struct{} {
					i := NewInjectionInfo()
					i.add(Feature{
						ftype:   OneAgent,
					})
					i.add(Feature{
						ftype:   DataIngest,
					})
					return i.features
				}(),
			},
			want: "data-ingest,oneagent",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &InjectionInfo{
				features: tt.fields.features,
			}
			if got := info.injectedAnnotation(); got != tt.want {
				t.Errorf("injectedAnnotation() = %v, want %v", got, tt.want)
			}
		})
	}
}
