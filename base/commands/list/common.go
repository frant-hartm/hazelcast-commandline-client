//go:build std || list

package list

import (
	"context"
	"fmt"

	"github.com/hazelcast/hazelcast-go-client"

	"github.com/hazelcast/hazelcast-commandline-client/base"
	"github.com/hazelcast/hazelcast-commandline-client/base/commands"
	"github.com/hazelcast/hazelcast-commandline-client/clc"
	"github.com/hazelcast/hazelcast-commandline-client/clc/cmd"
	"github.com/hazelcast/hazelcast-commandline-client/internal"
	"github.com/hazelcast/hazelcast-commandline-client/internal/output"
	"github.com/hazelcast/hazelcast-commandline-client/internal/plug"
	"github.com/hazelcast/hazelcast-commandline-client/internal/proto/codec"
	"github.com/hazelcast/hazelcast-commandline-client/internal/serialization"
)

func getList(ctx context.Context, ec plug.ExecContext, sp clc.Spinner) (*hazelcast.List, error) {
	name := ec.Props().GetString(base.FlagName)
	ci, err := cmd.ClientInternal(ctx, ec, sp)
	if err != nil {
		return nil, err
	}
	sp.SetText(fmt.Sprintf("Getting List '%s'", name))
	return ci.Client().GetList(ctx, name)
}

func removeFromList(ctx context.Context, ec plug.ExecContext, name string, index int32, valueStr string) error {
	rowV, stop, err := ec.ExecuteBlocking(ctx, func(ctx context.Context, sp clc.Spinner) (any, error) {
		indexCall := valueStr == ""
		ci, err := cmd.ClientInternal(ctx, ec, sp)
		if err != nil {
			return nil, err
		}
		cmd.IncrementClusterMetric(ctx, ec, "total.list")
		pid, err := internal.StringToPartitionID(ci, name)
		if err != nil {
			return nil, err
		}
		sp.SetText(fmt.Sprintf("Removing value from List '%s'", name))
		var req *hazelcast.ClientMessage
		if indexCall {
			req = codec.EncodeListRemoveWithIndexRequest(name, index)
		} else {
			vd, err := commands.MakeValueData(ec, ci, valueStr)
			if err != nil {
				return nil, err
			}
			req = codec.EncodeListRemoveRequest(name, vd)
		}
		resp, err := ci.InvokeOnPartition(ctx, req, pid, nil)
		if err != nil {
			return nil, err
		}
		var vt int32
		var value any
		var colName string
		if indexCall {
			raw := codec.DecodeListRemoveWithIndexResponse(resp)
			vt = raw.Type()
			value, err = ci.DecodeData(raw)
			colName = "Removed Value"
			if err != nil {
				ec.Logger().Info("The value was not decoded, due to error: %s", err.Error())
				value = serialization.NondecodedType(serialization.TypeToLabel(vt))
			}
		} else {
			vt = serialization.TypeBool
			value = codec.DecodeListRemoveResponse(resp)
			colName = "Removed"
		}
		row := output.Row{
			output.Column{
				Name:  colName,
				Type:  vt,
				Value: value,
			},
		}
		if ec.Props().GetBool(base.FlagShowType) {
			row = append(row, output.Column{
				Name:  output.NameValueType,
				Type:  serialization.TypeString,
				Value: serialization.TypeToLabel(vt),
			})
		}
		return row, nil
	})
	if err != nil {
		return err
	}
	stop()
	msg := fmt.Sprintf("OK List '%s' was updated.\n", name)
	ec.PrintlnUnnecessary(msg)
	row := rowV.(output.Row)
	return ec.AddOutputRows(ctx, row)
}

func convertDataToRow(ci *hazelcast.ClientInternal, name string, data hazelcast.Data, showType bool) (output.Row, error) {
	vt := data.Type()
	value, err := ci.DecodeData(data)
	if err != nil {
		return nil, err
	}
	row := output.Row{
		output.Column{
			Name:  name,
			Type:  vt,
			Value: value,
		},
	}
	if showType {
		row = append(row, output.Column{
			Name:  output.NameValueType,
			Type:  serialization.TypeString,
			Value: serialization.TypeToLabel(vt),
		})
	}
	return row, nil
}
