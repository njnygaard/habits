package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Log struct {
	id     int
	habit  string
	logged time.Time
}

type Track struct {
	id        int
	habit     string
	eventDate time.Time
	started   bool
}

var db *sql.DB

func main() {
	// Create the root command
	rootCmd := &cobra.Command{
		Use:   "habits",
		Short: "Mark off your habits daily.",
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				return
			}
			os.Exit(0)
		},
	}

	// Initialize Viper
	if err := initConfig(); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}

	// Initialize SQLite database
	if err := initDB(); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {

		}
	}(db)

	rootCmd.AddCommand(&cobra.Command{
		Use:   "reset",
		Short: "Reset today.",
		Run: func(cmd *cobra.Command, args []string) {
			if err := resetToday(); err != nil {
				logrus.Error(err.Error())
				logrus.Error("Something broke.")
				os.Exit(1)
			}
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "today",
		Short: "What have you done today?",
		Run: func(cmd *cobra.Command, args []string) {
			if err := todaysHabits(); err != nil {
				logrus.Error(err.Error())
				logrus.Error("Something broke.")
				os.Exit(1)
			}
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "track",
		Short: "Add a Habit to track.",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				logrus.Error("Habit name required.")
				os.Exit(1)
			}
			if err := trackHabit(args[0]); err != nil {
				if strings.Contains(err.Error(), "habit exists") {
					logrus.Errorf("Already tracking %s.", args[0])
				} else {
					logrus.Error(err.Error())
					logrus.Error("Something broke.")
				}
				os.Exit(1)
			}
			logrus.Infof("Now tracking %s as a Habit!", args[0])
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "untrack",
		Short: "Stop tracking a Habit.",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				logrus.Error("Habit name required.")
				os.Exit(1)
			}
			if err := untrackHabit(args[0]); err != nil {
				if strings.Contains(err.Error(), "habit does not exist") {
					logrus.Errorf("Already not tracking %s.", args[0])
				} else {
					logrus.Error(err.Error())
					logrus.Error("Something broke.")
				}
				os.Exit(1)
			}
			logrus.Infof("No longer tracking %s as a Habit!", args[0])
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all current Habits.",
		Run: func(cmd *cobra.Command, args []string) {
			if err := listTrackedHabits(); err != nil {
				//if strings.Contains(err.Error(), "habit does not exist") {
				//	logrus.Errorf("Already not tracking %s.", args[0])
				//} else {
				//	logrus.Error(err.Error())
				//	logrus.Error("Something broke.")
				//}
				os.Exit(1)
			}
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "log",
		Short: "Log your Habits.",
		Run: func(cmd *cobra.Command, args []string) {
			if err := logHabits(); err != nil {
				if err.Error() == "user aborted" {
					logrus.Info("No changes made.")
					os.Exit(0)
				}

				logrus.Error(err.Error())
				logrus.Error("Something broke.")
				os.Exit(1)
			}
		},
	})

	if err := rootCmd.Execute(); err != nil {
		logrus.Errorf("Error executing command: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() error {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	return nil
}

// initDB initializes the SQLite database connection.
func initDB() error {
	var err error
	dbPath := viper.GetString("database.path")
	if dbPath == "" {
		dbPath = "habits.db" // default database path
	}

	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("error opening database: %w", err)
	}

	createLogSQL := `
		CREATE TABLE IF NOT EXISTS log (
			id INTEGER PRIMARY KEY AUTOINCREMENT
		,	habit TEXT NOT NULL
		,	logged DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`

	_, err = db.Exec(createLogSQL)
	if err != nil {
		return fmt.Errorf("error creating table: %w", err)
	}

	createTrackSQL := `
		CREATE TABLE IF NOT EXISTS track (
			id INTEGER PRIMARY KEY AUTOINCREMENT
		,	habit TEXT NOT NULL
		,	eventDate DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		,	started bool NOT NULL DEFAULT true
		);`

	_, err = db.Exec(createTrackSQL)
	if err != nil {
		return fmt.Errorf("error creating table: %w", err)
	}

	return nil
}

func getTracks() (tracks []Track, err error) {
	selectExistingHabitsSQL := `
		select *
		from track
		order by eventDate asc;
		`
	rows, err := db.Query(selectExistingHabitsSQL)
	if err != nil {
		logrus.Error("query failed", err)
		return
	}

	for rows.Next() {
		var t Track
		if err = rows.Scan(&t.id, &t.habit, &t.eventDate, &t.started); err != nil {
			return
		}
		tracks = append(tracks, t)

	}
	if err = rows.Err(); err != nil {
		return
	}
	return
}

func activeHabits(tracks []Track) (m map[string]bool, err error) {
	m = make(map[string]bool)

	for _, t := range tracks {
		if t.started {
			m[t.habit] = true
		} else {
			delete(m, t.habit)
		}
	}

	if len(m) == 0 {
		err = errors.New("no active habits")
	}
	return
}

func listTrackedHabits() error {
	var tracks []Track
	tracks, err := getTracks()
	if err != nil {
		return err
	}

	m, err := activeHabits(tracks)
	if err != nil {
		logrus.Warn("Not tracking any Habits yet!")
	}

	if len(m) == 0 {
		logrus.Warn("Not tracking any Habits yet!")
	}

	for k := range m {
		logrus.Infof("%s", k)
	}
	return nil
}

func trackHabit(name string) error {
	var tracks []Track
	tracks, err := getTracks()
	if err != nil {
		return err
	}

	m, _ := activeHabits(tracks)

	if m[name] {
		return errors.New("habit exists")
	}

	_, err = db.Exec(`
	INSERT INTO track(habit, started) VALUES (?)
	`, name, true)
	if err != nil {
		return err
	}

	return nil
}

func untrackHabit(name string) error {

	var tracks []Track
	tracks, err := getTracks()
	if err != nil {
		return err
	}

	m, _ := activeHabits(tracks)

	if !m[name] {
		return errors.New("habit does not exist")
	}

	_, err = db.Exec(`
	INSERT INTO track(habit, started) VALUES (?,?)
	`, name, false)
	if err != nil {
		return err
	}

	return nil
}

func logHabits() (err error) {
	var selections []string
	var tracks []Track

	tracks, err = getTracks()
	if err != nil {
		return
	}

	m, err := activeHabits(tracks)
	if err != nil {
		return
		//logrus.Warn("Not tracking any Habits yet!")
	}
	for k := range m {
		m[k] = false
	}

	selectTodaysHabitsSQL := `
		select *
		from log
		where date(logged) = date('now')
		order by logged asc;
		`
	rows, err := db.Query(selectTodaysHabitsSQL)
	if err != nil {
		//logrus.Error("query failed", err)
		return
	}

	for rows.Next() {
		var t Log
		if err = rows.Scan(&t.id, &t.habit, &t.logged); err != nil {
			return err
		}
		m[t.habit] = true

	}
	if err = rows.Err(); err != nil {
		return
	}

	//if len(m) == 0 {
	//	logrus.Warn("Not tracking any Habits yet!")
	//	return nil
	//}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Habits").
				OptionsFunc(func() []huh.Option[string] {
					var opts []huh.Option[string]
					for k := range m {
						if m[k] {
							opts = append(opts, huh.NewOption[string](k, k).Selected(true))
						} else {
							opts = append(opts, huh.NewOption[string](k, k).Selected(false))
						}
					}
					return opts
				}, nil).
				Value(&selections),
		),
	)
	err = form.Run()
	if err != nil {
		return err
	}

	//logrus.Info(selections)

	for _, v := range selections {
		_, err = db.Exec(`
			INSERT INTO log(habit) VALUES (?)
			`, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func todaysHabits() error {
	var tracks []Track
	tracks, err := getTracks()
	if err != nil {
		return err
	}

	m, err := activeHabits(tracks)
	if err != nil {
		logrus.Warn("Not tracking any Habits yet!")
	}
	for k := range m {
		m[k] = false
	}

	selectTodaysHabitsSQL := `
		select *
		from log
		where date(logged) = date('now')
		order by logged asc;
		`
	rows, err := db.Query(selectTodaysHabitsSQL)
	if err != nil {
		logrus.Error("query failed", err)
		return err
	}

	for rows.Next() {
		var t Log
		if err = rows.Scan(&t.id, &t.habit, &t.logged); err != nil {
			return err
		}
		m[t.habit] = true

	}
	if err = rows.Err(); err != nil {
		return err
	}

	for k := range m {
		if m[k] {
			fmt.Printf("🟩 %s\n", k)
		} else {
			fmt.Printf("⬛ %s\n", k)
		}
	}
	return nil
}

func resetToday() error {
	resetTodaysHabitsSQL := `
		delete
		from log
		where date(logged) = date('now');
		`
	_, err := db.Exec(resetTodaysHabitsSQL)
	if err != nil {
		return fmt.Errorf("error deleting from table: %w", err)
	}
	return nil
}
