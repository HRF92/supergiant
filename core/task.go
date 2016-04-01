package core

import (
	"encoding/json"
	"log"
	"path"
	"strconv"

	"github.com/supergiant/supergiant/types"
)

type TaskCollection struct {
	core *Core
}

type TaskResource struct {
	collection *TaskCollection
	*types.Task

	ID types.ID `json:"id"`
}

// NOTE this does not inherit from types like model does; all we need is a List
// object, internally, that has a slice of our composed model above.
type TaskList struct {
	Items []*TaskResource `json:"items"`
}

const (
	statusQueued  = "QUEUED"
	statusRunning = "RUNNING"
	statusFailed  = "FAILED"
)

// EtcdKey implements the Collection interface.
func (c *TaskCollection) EtcdKey(id types.ID) string {
	key := "/tasks"
	if id != nil {
		key = path.Join(key, *id)
	}
	return key
}

// InitializeResource implements the Collection interface.
func (c *TaskCollection) InitializeResource(r Resource) {
	resource := r.(*TaskResource)
	resource.collection = c
}

// List returns an TaskList.
func (c *TaskCollection) List() (*TaskList, error) {
	list := new(TaskList)
	err := c.core.DB.ListInOrder(c, list)
	return list, err
}

// New initializes an Task with a pointer to the Collection.
func (c *TaskCollection) New() *TaskResource {
	return &TaskResource{
		Task: &types.Task{
			Meta: types.NewMeta(),
		},
	}
}

// Create takes an Task and creates it in etcd. It also creates a Kubernetes
// Namespace with the name of the Task.
func (c *TaskCollection) Create(r *TaskResource) (*TaskResource, error) {
	if err := c.core.DB.CreateInOrder(c, r); err != nil {
		return nil, err
	}
	return r, nil
}

// Get takes a name and returns an TaskResource if it exists.
func (c *TaskCollection) Get(id types.ID) (*TaskResource, error) {
	r := c.New()
	if err := c.core.DB.Get(c, id, r); err != nil {
		return nil, err
	}

	// NOTE have to set ID since it is autogenerated, and the Get() method does
	// not handle parsing keys like the other _InOrder methods do
	r.ID = id

	return r, nil
}

// NOTE kinda like a New().Save()
func (c *TaskCollection) Start(t types.TaskType, msg interface{}) (*TaskResource, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	task := &TaskResource{
		Task: &types.Task{
			Type:        t,
			Data:        data,
			Status:      statusQueued,
			MaxAttempts: 20, // TODO may want to expose this as an arg
		},
	}
	return c.Create(task)
}

// Resource-level
//==============================================================================

// PersistableObject satisfies the Resource interface
func (r *TaskResource) PersistableObject() interface{} {
	return r.Task
}

// Delete deletes the Task in etcd.
func (r *TaskResource) Delete() error {
	return r.collection.core.DB.Delete(r.collection, r.ID)
}

// Save saves the Task in etcd through an update.
func (r *TaskResource) Save() error {
	return r.collection.core.DB.Update(r.collection, r.ID, r)
}

// Implements OrderedModel interface
func (r *TaskResource) SetID(id types.ID) {
	r.ID = id
}

func (r *TaskResource) IsQueued() bool {
	return r.Status == statusQueued
}

// Claim updates the Task status to "RUNNING" and returns nil. CompareAndSwap is
// used to prevent a race condition and ensure only one worker performs the task.
func (r *TaskResource) Claim() error {
	prev := r
	// next := *r
	// next.Status = statusRunning

	// NOTE we have to do this instead of the above, because nested pointers are
	// not de-referenced.
	t := *r.Task
	t.Status = statusRunning
	next := &TaskResource{Task: &t}

	return r.collection.core.DB.CompareAndSwap(r.collection, r.ID, prev, next)
}

func (r *TaskResource) RecordError(err error) error {
	log.Println(err)

	r.Error = err.Error()
	if r.Attempts < r.MaxAttempts {
		r.Status = statusQueued // Add back to queue for retry
	} else {
		r.Status = statusFailed // TODO failed tasks will build up in the queue
	}
	r.Attempts++

	return r.Save()
}

func (r *TaskResource) TypeName() string {
	switch r.Type {
	case types.TaskTypeDeleteApp:
		return "DeleteApp"
	case types.TaskTypeDeleteComponent:
		return "DeleteComponent"
	case types.TaskTypeDeployComponent:
		return "DeployComponent"
	case types.TaskTypeStartInstance:
		return "StartInstance"
	case types.TaskTypeStopInstance:
		return "StopInstance"
	default:
		return strconv.Itoa(int(r.Type))
	}
}
