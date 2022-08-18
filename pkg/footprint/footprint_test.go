package footprint

import (
	_ "embed"
	"testing"
	"time"
)

func Test_readEC2Instances(t *testing.T) {
	err := readEC2Instances()
	if err != nil {
		t.Errorf("readEC2Instances() error = %v", err)
	}

	tests := []struct {
		instanceType string
		value        EC2Instance
	}{
		{
			instanceType: "m5d.16xlarge",
			value: EC2Instance{
				PowerAt50Percent:             451.9,
				ManufacturingEmissionsHourly: 38.8,
			},
		},
		{
			instanceType: "t2.micro",
			value: EC2Instance{
				PowerAt50Percent:             4.9,
				ManufacturingEmissionsHourly: 0.9,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.instanceType, func(t *testing.T) {
			if ec2instances[tt.instanceType] != tt.value {
				t.Errorf("readEC2Instances() instance type %s - want value %v, got value %v", tt.instanceType, tt.value, ec2instances[tt.instanceType])
			}
		})
	}
}

func Test_readAWSRegions(t *testing.T) {
	err := readAWSRegions()
	if err != nil {
		t.Errorf("readAWSRegions() error = %v", err)
	}

	tests := []struct {
		regionCode string
		awsRegion  AWSRegion
	}{
		{regionCode: "eu-central-1", awsRegion: AWSRegion{CarbonIntensity: 338, PUE: 1.2}},
		{regionCode: "eu-west-1", awsRegion: AWSRegion{CarbonIntensity: 316, PUE: 1.2}},
		{regionCode: "us-east-1", awsRegion: AWSRegion{CarbonIntensity: 415.755, PUE: 1.2}},
	}
	for _, tt := range tests {
		t.Run(tt.regionCode, func(t *testing.T) {
			if awsRegions[tt.regionCode] != tt.awsRegion {
				t.Errorf("readAWSRegions() code %s - want value %v, got value %v", tt.regionCode, tt.awsRegion, awsRegions[tt.regionCode])
			}
		})
	}
}

func TestCarbonIntensity(t *testing.T) {
	type args struct {
		regionCode string
	}

	tests := []struct {
		name    string
		args    args
		want    float64
		wantErr bool
	}{
		{name: "eu-central-1", args: args{"eu-central-1"}, want: 338, wantErr: false},
		{name: "ap-southeast-2", args: args{"ap-southeast-2"}, want: 790, wantErr: false},
		{name: "unknown", args: args{"unknown"}, want: 0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CarbonIntensity(tt.args.regionCode)
			if (err != nil) != tt.wantErr {
				t.Errorf("CarbonIntensity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CarbonIntensity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPUE(t *testing.T) {
	type args struct {
		regionCode string
	}
	tests := []struct {
		name    string
		args    args
		want    float64
		wantErr bool
	}{
		{name: "eu-central-1", args: args{"eu-central-1"}, want: 1.2, wantErr: false},
		{name: "ap-southeast-2", args: args{"ap-southeast-2"}, want: 1.2, wantErr: false},
		{name: "unknown", args: args{"unknown"}, want: 0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PUE(tt.args.regionCode)
			if (err != nil) != tt.wantErr {
				t.Errorf("PUE() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PUE() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPowerAt50Percent(t *testing.T) {
	type args struct {
		ec2InstanceType string
	}
	tests := []struct {
		name    string
		args    args
		want    float64
		wantErr bool
	}{
		{name: "a1.medium", args: args{"a1.medium"}, want: 3.2, wantErr: false},
		{name: "c3.8xlarge", args: args{"c3.8xlarge"}, want: 191.1, wantErr: false},
		{name: "m5.2xlarge", args: args{"m5.2xlarge"}, want: 56.5, wantErr: false},
		{name: "t2.micro", args: args{"t2.micro"}, want: 4.9, wantErr: false},
		{name: "unknown", args: args{"unknown"}, want: 0, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PowerAt50Percent(tt.args.ec2InstanceType)
			if (err != nil) != tt.wantErr {
				t.Errorf("PowerAt50Percent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PowerAt50Percent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManufacturingEmissions(t *testing.T) {
	type args struct {
		ec2InstanceType string
	}
	tests := []struct {
		name    string
		args    args
		want    float64
		wantErr bool
	}{
		{name: "a1.medium", args: args{"a1.medium"}, want: 1.8, wantErr: false},
		{name: "c3.8xlarge", args: args{"c3.8xlarge"}, want: 31.5, wantErr: false},
		{name: "m5.2xlarge", args: args{"m5.2xlarge"}, want: 3.9, wantErr: false},
		{name: "t2.micro", args: args{"t2.micro"}, want: 0.9, wantErr: false},
		{name: "unknown", args: args{"unknown"}, want: 0, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ManufacturingEmissions(tt.args.ec2InstanceType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ManufacturingEmissions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ManufacturingEmissions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAWS(t *testing.T) {
	type args struct {
		regionCode   string
		instanceType string
		duration     time.Duration
	}

	tests := []struct {
		name    string
		args    args
		want    float64
		wantErr bool
	}{
		{name: "zero duration", args: args{"eu-west-1", "t2.micro", 0 * time.Hour}, want: 0, wantErr: false},
		{name: "unknown region", args: args{"unknown", "t2.micro", time.Hour}, want: 0, wantErr: true},
		{name: "unknown instance", args: args{"eu-west-1", "unknown", time.Hour}, want: 0, wantErr: true},
		{name: "eu-west-1 t2.micro 1 hour", args: args{"eu-west-1", "t2.micro", time.Hour}, want: 2.75808, wantErr: false},
		{name: "ap-southeast-2 c4.large 1 hour", args: args{"ap-southeast-2", "c4.large", time.Hour}, want: 14.7824, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AWS(tt.args.regionCode, tt.args.instanceType, tt.args.duration)
			if (err != nil) != tt.wantErr {
				t.Errorf("AWS() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("AWS() = %v, want %v", got, tt.want)
			}
		})
	}
}
