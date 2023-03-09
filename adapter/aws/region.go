package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"github.com/olekukonko/tablewriter"

)

// RegionSummary is a structure holding deployed instances in a given region
type RegionSummary struct {
	Region    string
	Instances []Instance
}

// AccountSummary is a structure holding a slice of regions summaries across an entire account
type AccountSummary []RegionSummary

// Print prints the summary of instances in an account in tabular format
func (u AccountSummary) Print() {
	for _, region := range u {
		region.Print()
		fmt.Println("")
	}
}

// Prompts user for confirmation
func (u AccountSummary) Prompt(action string) []string {
	var instances []Instance
	var instanceIDs []string
	var s string

	questionLabel := "\n" + "This command will " + action + " the following running instances matching the filter:\n"
	confirmationLabel := "\nWould you like to proceed? [Y/n]"
	errLabel := "No instances are available for " + action + " command"
	for _, region := range u {
		instances = region.Prompt(instances, action)
	}

	//print labels onto terminal and scan terminal for input
	if len(instances) == 0 {
		fmt.Print(errLabel)
	} else {
		fmt.Println(questionLabel)
		WriteTable(instances)
		fmt.Println(confirmationLabel)
	}

	fmt.Scanln(&s)
	for _, instance := range instances {
		instanceIDs = append(instanceIDs,instance.ID)
	}
	//if user acknowledges, return instanceIDs associated
	if s == "Y" {
		return instanceIDs
	}
	//else, return empty
	return []string{}
}

// GetInstanceRegion returns the region of an instance given an account summary
func GetInstanceRegion(accSum AccountSummary, id string) (string, error) {
	for _, region := range accSum {
		for _, instance := range region.Instances {
			if instance.ID == id {
				return region.Region, nil
			}
		}
	}
	return "", errors.New("instance not found")
}

// Print prints the summary of instances in a given region in tabular format
func (u RegionSummary) Print() {
	fmt.Println(u.Region)
	WriteTable(u.Instances)
}

// GetRegions is a function to retrieve all active regions in an account
func GetRegions() (regions []string) {

	ctx := context.TODO()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}
	svc := ec2.NewFromConfig(cfg)
	input := &ec2.DescribeRegionsInput{
		Filters: []types.Filter{
			{
				Name: aws.String("opt-in-status"),
				Values: []string{
					"opt-in-not-required",
				},
			},
		},
	}

	result, err := svc.DescribeRegions(ctx, input)
	if err != nil {
		var ae smithy.APIError
		if errors.As(err, &ae) {
			log.Printf("code: %s, message: %s, fault: %s", ae.ErrorCode(), ae.ErrorMessage(), ae.ErrorFault().String())
		}
		return
	}

	for _, r := range result.Regions {
		regions = append(regions, *r.RegionName)
	}

	return regions
}

func (u RegionSummary) Prompt(instances []Instance, action string) ([]Instance) {
	const STOP string = "stop"
	const START string = "start"
	for _, instance := range u.Instances {
		switch action {
		case STOP:
			if instance.Status == types.InstanceStateNameRunning {
				instances = append(instances, instance)
			}
		case START:
			if instance.Status == types.InstanceStateNameStopped {
				instances = append(instances, instance)
			}
		}
	}
	return instances
}



func WriteTable(data []Instance) {
	var header []string
	var headerColors []tablewriter.Colors

	table := tablewriter.NewWriter(os.Stdout) 

	structFields := reflect.VisibleFields(reflect.TypeOf(data[0]))
	for _, f := range structFields {
		header = append(header, f.Name)
		headerColors = append(headerColors, tablewriter.Colors{tablewriter.Bold})
	}
	table.SetHeader(header)
	table.SetHeaderColor(headerColors...)

	for _, o := range data {
		var row []string
		var rowColor []tablewriter.Colors
		for _, f := range structFields {
			value := fmt.Sprintf("%v", reflect.ValueOf(o).FieldByName(f.Name).Interface())
			row = append(row, value)
			switch f.Name {
			case "Name":
				rowColor = append(rowColor, tablewriter.Colors{tablewriter.Bold})
			case "Status":
				switch o.Status {
				case types.InstanceStateNameRunning:
					rowColor = append(rowColor, tablewriter.Colors{tablewriter.FgGreenColor})
				case types.InstanceStateNameStopped:
					rowColor = append(rowColor, tablewriter.Colors{tablewriter.FgRedColor})
				case types.InstanceStateNamePending, types.InstanceStateNameStopping:
					rowColor = append(rowColor, tablewriter.Colors{tablewriter.FgYellowColor})
				case types.InstanceStateNameTerminated:
					rowColor = append(rowColor, tablewriter.Colors{tablewriter.FgBlackColor})
				default:
					rowColor = append(rowColor, tablewriter.Colors{})
				}
			default:
				rowColor = append(rowColor, tablewriter.Colors{})
			}
		}
		table.Rich(row,rowColor)
	}

	table.Render()

}