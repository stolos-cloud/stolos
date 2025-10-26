package job

import (
	"fmt"
	"log"
	"reflect"

	"github.com/NVIDIA/gontainer/v2"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type JobService struct {
	rs    *gontainer.Resolver
	sched gocron.Scheduler

	jobRegistry map[string]*StolosJob
}

type StolosJob struct {
	Name       string
	Definition gocron.JobDefinition
	JobFunc    interface{} // any function signature
	JobArgs    []any       // argument specs or placeholders
	Options    []gocron.JobOption

	JobID uuid.UUID
	Job   gocron.Job
}

func NewJobService(rs *gontainer.Resolver) (*JobService, error) {

	s, err := gocron.NewScheduler()
	if err != nil {
		log.Print("Failed to start scheduler:", err)
		return nil, err
	}

	log.Printf("Started JobService Scheduler.")
	svc := &JobService{
		rs:          rs,
		sched:       s,
		jobRegistry: make(map[string]*StolosJob),
	}

	svc.RegisterJobs(
		ClusterHealthCheckJob,
		NodeInfoReconciler,
		NodeStatusUpdateJob,
	)

	return svc, nil
}

func (s *JobService) Start() {
	s.sched.Start()
}

func (s *JobService) RegisterJob(job *StolosJob) (*StolosJob, error) {

	// Validate input job before doing reflection
	//if err := job.Validate(s.rs); err != nil {
	//	return nil, fmt.Errorf("validation failed for job %q: %w", job.Name, err)
	//}

	wrappedJobFunc := func() {
		log.Printf("[JobService] Starting job %s\n", job.Name)

		// Resolve args dynamically using gontainer
		args := make([]reflect.Value, len(job.JobArgs))
		for i, arg := range job.JobArgs {
			if arg == nil {
				args[i] = reflect.Zero(reflect.TypeOf((*any)(nil)).Elem())
				continue
			}

			typ := reflect.TypeOf(arg).Elem() // e.g. *talos.TalosService → talos.TalosService
			ptr := reflect.New(typ)           // → *talos.TalosService
			ptrPtr := reflect.New(ptr.Type()) // → **talos.TalosService

			// Put ptr inside ptrPtr (so we can take its address)
			ptrPtr.Elem().Set(ptr)

			// Call resolver with &ptr (type **T)
			if err := s.rs.Resolve(ptrPtr.Interface()); err != nil {
				log.Printf("[JobService] Failed to resolve arg %v: %v", typ, err)
				return
			}

			// Deref to get resolved value (*T)
			args[i] = reflect.ValueOf(ptrPtr.Elem().Interface())
		}

		// Invoke the JobFunc via reflection
		fval := reflect.ValueOf(job.JobFunc)
		fval.Call(args)

		log.Printf("[JobService] Finished job %s\n", job.Name)
	}

	job.Options = append(job.Options, gocron.WithName(job.Name))

	cronJob, err := s.sched.NewJob(
		job.Definition,
		gocron.NewTask(wrappedJobFunc),
		job.Options...,
	)
	if err != nil {
		return nil, err
	}

	job.JobID = cronJob.ID()
	job.Job = cronJob
	s.jobRegistry[job.Name] = job

	log.Printf("[JobService] Registered job %s (ID: %s)", job.Name, job.JobID)
	return job, nil
}

func (s *JobService) RegisterJobs(jobs ...*StolosJob) {
	for _, job := range jobs {
		_, err := s.RegisterJob(job)
		if err != nil {
			log.Print("[RegisterAllJobs] Failed to register job:", err)
		}
	}
}

func (s *JobService) UnregisterJob(name string) error {
	if job, ok := s.jobRegistry[name]; ok {
		err := s.sched.RemoveJob(job.JobID)
		if err != nil {
			return errors.Wrap(err, "failed to unregister job")
		}
		s.jobRegistry[name] = nil
		return nil
	} else {
		return fmt.Errorf("job %s not found", name)
	}
}

func (s *JobService) GetJob(name string) (*StolosJob, error) {
	if job, ok := s.jobRegistry[name]; ok {
		return job, nil
	} else {
		return nil, fmt.Errorf("job %s not found", name)
	}
}

// GetScheduler returns the underlying scheduler. adding a job directly will not register it with the job service.
func (s *JobService) GetScheduler() gocron.Scheduler {
	return s.sched
}

func (s *JobService) invokeJobFunc(job *StolosJob) error {
	if job.JobFunc == nil {
		return fmt.Errorf("job %q has nil JobFunc", job.Name)
	}

	fv := reflect.ValueOf(job.JobFunc)
	if fv.Kind() != reflect.Func {
		return fmt.Errorf("job %q JobFunc is not a function", job.Name)
	}

	ft := fv.Type()
	if ft.NumIn() != len(job.JobArgs) {
		return fmt.Errorf(
			"job %q: JobFunc expects %d args, but JobArgs has %d",
			job.Name, ft.NumIn(), len(job.JobArgs),
		)
	}

	args := make([]reflect.Value, len(job.JobArgs))
	for i, arg := range job.JobArgs {
		if arg == nil {
			args[i] = reflect.Zero(ft.In(i))
			continue
		}

		argType := reflect.TypeOf(arg)
		if argType.Kind() == reflect.Ptr {
			// Attempt DI resolution from gontainer
			val := reflect.New(argType.Elem()).Interface()
			if err := s.rs.Resolve(val); err != nil {
				return fmt.Errorf(
					"job %q: failed to resolve dependency %s: %w",
					job.Name, argType, err,
				)
			}
			args[i] = reflect.ValueOf(val)
		} else {
			// Literal argument, use as-is
			args[i] = reflect.ValueOf(arg)
		}
	}

	fv.Call(args)
	return nil
}

func (s *JobService) ExecuteJobSync(name string) error {
	job, ok := s.jobRegistry[name]
	if !ok {
		return fmt.Errorf("job %s not found", name)
	}

	log.Printf("[JobService] Executing job %s synchronously", name)
	return s.invokeJobFunc(job)
}

func (s *JobService) ExecuteJobAsync(name string) error {
	job, ok := s.jobRegistry[name]
	if !ok {
		return fmt.Errorf("job %s not found", name)
	}

	log.Printf("[JobService] Executing job %s asynchronously", name)
	go func() {
		if err := s.invokeJobFunc(job); err != nil {
			log.Printf("[JobService] Async job %s failed: %v", name, err)
		}
	}()

	return nil
}
