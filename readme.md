# Habits

## Development Notes

You can't make a SQL identifier from a prepared statement.
So I can't make a table for each habit.
Seems like I should just keep a log of all the habits.

I want to be able to identify when we start tracking a habit and when we don't want to track it anymore.
That can probably just be a running log for each habit.
That means just an entry in the table counts as a 'track'.
Additionally, I will need another table that has entries in it for when we started tracking that habit.
That will give me the information for which days are empty as opposed to not tracked.