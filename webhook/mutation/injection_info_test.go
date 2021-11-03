package mutation

import (
	"testing"
)

func TestFeature_annotationValue(t *testing.T) {
	type fields struct {
		Enabled bool
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "",
			fields: fields{
				Enabled: true,
			},
			want: "true",
		},
		{
			name: "",
			fields: fields{
				Enabled: false,
			},
			want: "false",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Feature{
				ftype:   0,
				Enabled: tt.fields.Enabled,
			}
			if got := f.annotationValue(); got != tt.want {
				t.Errorf("annotationValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFeature_name(t *testing.T) {
	type fields struct {
		ftype   FeatureType
		Enabled bool
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
				Enabled: false,
			},
			want: "oneagent",
		},
		{
			name: "",
			fields: fields{
				ftype:   DataIngest,
				Enabled: false,
			},
			want: "data-ingest",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Feature{
				ftype:   tt.fields.ftype,
				Enabled: tt.fields.Enabled,
			}
			if got := f.name(); got != tt.want {
				t.Errorf("name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInjectionInfo_enabled(t *testing.T) {
	features := func() map[*Feature]struct{} {
		i := NewInjectionInfo()
		i.add(&Feature{
			ftype:   OneAgent,
			Enabled: true,
		})
		i.add(&Feature{
			ftype:   DataIngest,
			Enabled: false,
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
			want: false,
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
				t.Errorf("in() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInjectionInfo_injectedAnnotation(t *testing.T) {
	type fields struct {
		features map[*Feature]struct{}
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "OA-enabled",
			fields: fields{
				features: func() map[*Feature]struct{} {
					i := NewInjectionInfo()
					i.add(&Feature{
						ftype:   OneAgent,
						Enabled: true,
					})
					return i.features
				}(),
			},
			want: "oneagent",
		},
		{
			name: "OA enabled and DI explicitly disabled",
			fields: fields{
				features: func() map[*Feature]struct{} {
					i := NewInjectionInfo()
					i.add(&Feature{
						ftype:   OneAgent,
						Enabled: true,
					})
					i.add(&Feature{
						ftype:   DataIngest,
						Enabled: false,
					})
					return i.features
				}(),
			},
			want: "oneagent",
		},
		{
			name: "OA and DI enabled",
			fields: fields{
				features: func() map[*Feature]struct{} {
					i := NewInjectionInfo()
					i.add(&Feature{
						ftype:   OneAgent,
						Enabled: true,
					})
					i.add(&Feature{
						ftype:   DataIngest,
						Enabled: true,
					})
					return i.features
				}(),
			},
			want: "data-ingest,oneagent",
		},
		{
			name: "OA and DI explicitly disabled",
			fields: fields{
				features: func() map[*Feature]struct{} {
					i := NewInjectionInfo()
					i.add(&Feature{
						ftype:   OneAgent,
						Enabled: false,
					})
					i.add(&Feature{
						ftype:   DataIngest,
						Enabled: false,
					})
					return i.features
				}(),
			},
			want: "",
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
