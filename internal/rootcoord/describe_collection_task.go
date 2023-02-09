package rootcoord

import (
	"context"

	"github.com/milvus-io/milvus-proto/go-api/commonpb"
	"github.com/milvus-io/milvus-proto/go-api/milvuspb"
	"github.com/milvus-io/milvus/internal/util/typeutil"
)

// describeCollectionTask describe collection request task
type describeCollectionTask struct {
	baseTask
	Req              *milvuspb.DescribeCollectionRequest
	Rsp              *milvuspb.DescribeCollectionResponse
	allowUnavailable bool
}

func (t *describeCollectionTask) Prepare(ctx context.Context) error {
	t.SetStep(typeutil.TaskStepPreExecute)
	if err := CheckMsgType(t.Req.Base.MsgType, commonpb.MsgType_DescribeCollection); err != nil {
		return err
	}
	return nil
}

// Execute task execution
func (t *describeCollectionTask) Execute(ctx context.Context) (err error) {
	t.SetStep(typeutil.TaskStepExecute)
	coll, err := t.core.describeCollection(ctx, t.Req, t.allowUnavailable)
	if err != nil {
		return err
	}
	aliases := t.core.meta.ListAliasesByID(coll.CollectionID)
	t.Rsp = convertModelToDesc(coll, aliases)
	return nil
}
