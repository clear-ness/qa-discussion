package app

import (
	"net/http"
	"strings"
	"time"

	"github.com/clear-ness/qa-discussion/mlog"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/utils"
	"github.com/jasonlvhit/gocron"
)

const (
	GetUsersLimit = 100
)

type EmailBatchingJob struct {
	server   *Server
	interval *string
}

func NewEmailBatchingJob(s *Server, interval *string) *EmailBatchingJob {
	return &EmailBatchingJob{
		server:   s,
		interval: interval,
	}
}

func (job *EmailBatchingJob) useCron() bool {
	if job.interval == nil {
		return true
	}

	return false
}

func (job *EmailBatchingJob) scheduleJobs() *model.AppError {
	if !*job.server.Config().EmailBatchJobSettings.Enable {
		return model.NewAppError("InitEmailBatching", "email_batching.app_error", nil, "", http.StatusInternalServerError)
	}

	if job.useCron() {
		if *job.server.Config().EmailBatchJobSettings.ThreeHourly {
			job.scheduleThreeHourlyEmailBatchJobs()
		}

		if *job.server.Config().EmailBatchJobSettings.Daily {
			job.scheduleDailyEmailBatchJobs()
		}

		if *job.server.Config().EmailBatchJobSettings.Weekly {
			job.scheduleWeeklyEmailBatchJobs()
		}
	} else {
		// e.g. AWS ECS Scheduled Task
		job.emailBatchJob(*job.interval)
	}

	return nil
}

// TODO: use system table
func (job *EmailBatchingJob) emailBatchJob(inboxInterval string) {
	var past int64
	switch inboxInterval {
	case model.NOTIFICATION_INBOX_INTERVAL_THREE_HOUR:
		past = utils.MillisFromTime(time.Now().Add(-3 * time.Hour))
	case model.NOTIFICATION_INBOX_INTERVAL_DAY:
		past = utils.MillisFromTime(time.Now().Add(-24 * time.Hour))
	case model.NOTIFICATION_INBOX_INTERVAL_WEEK:
		past = utils.MillisFromTime(time.Now().Add(-24 * 7 * time.Hour))
	default:
		return
	}

	lastDoneUserId := strings.Repeat("0", 26)

	for {
		users, err := job.server.Store.User().GetByInboxInterval(lastDoneUserId, inboxInterval, GetUsersLimit)
		if err != nil || users == nil || len(users) <= 0 {
			if !job.useCron() {
				mlog.Info("self stopping job server...")

				job.server.Shutdown()
			}

			break
		}

		lastDoneUserId = users[len(users)-1].Id
		mlog.Info("email batch job last userId: ", mlog.String("userId", lastDoneUserId))

		for _, user := range users {
			if user.DeleteAt != 0 || user.Email == "" || !user.EmailVerified {
				continue
			}

			minDate := user.LastInboxMessageViewed
			if user.LastInboxMessageViewed < past {
				minDate = past
			}

			var count int64
			// TODO: teamを考慮
			if count, err = job.server.Store.InboxMessage().GetInboxMessagesUnreadCount(user.Id, minDate, ""); err != nil {
				continue
			}
			if count <= 0 {
				continue
			}

			var messages []*model.InboxMessage
			// TODO: teamを考慮
			if messages, err = job.server.Store.InboxMessage().GetInboxMessages(minDate, user.Id, ">", 0, 10, ""); err != nil {
				continue
			}
			if len(messages) <= 0 {
				continue
			}

			job.server.Go(func() {
				if err := SendInboxMessagesDigestEmail(user.Email, *job.server.Config().ServiceSettings.SiteURL, count, job.server.Config()); err != nil {
					mlog.Error("Failed to send inbox messages digest email", mlog.Err(err))
				}
			})
		}
	}
}

func (job *EmailBatchingJob) scheduleThreeHourlyEmailBatchJobs() {
	gocron.Every(1).Day().At("00:00").DoSafely(job.emailBatchJob, model.NOTIFICATION_INBOX_INTERVAL_THREE_HOUR)
	gocron.Every(1).Day().At("03:00").DoSafely(job.emailBatchJob, model.NOTIFICATION_INBOX_INTERVAL_THREE_HOUR)
	gocron.Every(1).Day().At("06:00").DoSafely(job.emailBatchJob, model.NOTIFICATION_INBOX_INTERVAL_THREE_HOUR)
	gocron.Every(1).Day().At("09:00").DoSafely(job.emailBatchJob, model.NOTIFICATION_INBOX_INTERVAL_THREE_HOUR)
	gocron.Every(1).Day().At("12:00").DoSafely(job.emailBatchJob, model.NOTIFICATION_INBOX_INTERVAL_THREE_HOUR)
	gocron.Every(1).Day().At("15:00").DoSafely(job.emailBatchJob, model.NOTIFICATION_INBOX_INTERVAL_THREE_HOUR)
	gocron.Every(1).Day().At("18:00").DoSafely(job.emailBatchJob, model.NOTIFICATION_INBOX_INTERVAL_THREE_HOUR)
	gocron.Every(1).Day().At("21:00").DoSafely(job.emailBatchJob, model.NOTIFICATION_INBOX_INTERVAL_THREE_HOUR)
}

func (job *EmailBatchingJob) scheduleDailyEmailBatchJobs() {
	gocron.Every(1).Day().At("07:30").DoSafely(job.emailBatchJob, model.NOTIFICATION_INBOX_INTERVAL_DAY)
}

func (job *EmailBatchingJob) scheduleWeeklyEmailBatchJobs() {
	gocron.Every(1).Monday().At("10:30").DoSafely(job.emailBatchJob, model.NOTIFICATION_INBOX_INTERVAL_WEEK)
}

func (job *EmailBatchingJob) startJobs() {
	if job.useCron() {
		// Start all the pending jobs
		<-gocron.Start()
	}
}

func (job *EmailBatchingJob) StopJobs() {
	if job.useCron() {
		gocron.Clear()
	}
}
