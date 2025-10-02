# Developer DB migration guide for Igor

These instructions cover what needs to be done when you need to make changes
to the database that will require upgrading an already existing Igor database.

### When You Need To Migrate
Basically, any time you make changes to the structure of the database. Even when GORM can do the job on its own, our migration scripts create a much more reliable way to validate that the database is up-to-date, not only by doing things that GORM is capable of but also by upgrading the internal version number on the database itself, which the Igor server checks at startup for safety and integrity reasons.

### How We Upgrade
We use a database migration tool called Atlas that understands both SQLite and GORM. Atlas runs locally on your machine to create files that can be executed to perform an in-place upgrade. Atlas itself doesn't need to be deployed with Igor. In the end we get a series of SQL files that does the actual migration and our own script 'migrate.go' that compiles into a binary that is run to execute those scripts in the prescribed order and increment the version number on the DB itself.

## Instructions
### Make a copy of your database WITHOUT changes to it.

This might be a bit of a bother if you've already been changing your database tables to work on your feature. If you've already run the igor-server with these changes to the tables structs in the code, GORM will have changed the tables.

To help with this, there are example db's in the

`db-migrate/test-dbs`

folder. Grab a copy of the one you want to use into the `.database `folder and rename it to be your current `igor.db` file.

There is only a bare minimum of entries in these test db files. If you need more data in them, check your most current code updates into another branch then roll back to an earlier version of Igor that matches the database version, build the server, and run it in with the `igor.db` file so you can populate your it with more reservations, etc., that will test your migration changes.

Once you have a testable `igor.db`, go back to your current code commits and rebuild/resume. _However, it is important to not actually run Igor against this database and possibly change it before running your migration procedure. Otherwise, you'll be repeating the entire process described above again._

If it helps, save off a copy of your `igor.db` file somewhere at this point so you if you accidentally update it you can resume from a clean copy. 

### Download and Install Atlas

Run the following at your shell prompt and follow the instructions:

`curl -sSf https://atlasgo.sh | sh`

From the top level of the Igor project folder run:

`go get -u ariga.io/atlas-provider-gorm`

you may also need to run

`go mod tidy`

This will create changes in Igor's `go.mod` file so it can inspect the server code that defines Igor's SQLite/GORM tables.

Find the file:

`igor2/internal/app/igor-server/tools.go`

and un-comment the import line for the Atlas package.

> ⚠️ **These changes are temporary! DO NOT commit them to the repository. See the notes below about what gets committed and what gets reverted!**

### Creating a Database Change File

At this point, you should probably run a quick compile test to make sure you are not missing anything. You should also have restored your commits with all the working changes that use the new GORM updates you've made to the table structs.

With all this in place, use your shell prompt to go to the `igor2/db-migrate` folder and run the command:

`atlas migrate diff --env gorm`

If all steps have been followed accurately up to this point, Atlas will generate a SQL file with a timestamp name that contains the database migrations step. Example:

`db-migrate/migrations/20250314234713.sql`

It will also change the `atlas.sum` file to record the new hashes generated for the file. We need to change this to match our own way of tracking the change files. Rename the SQL file to follow the pattern of other migration steps. For example, if this will be database version 3, to the following:

`mv 20250314234713.sql migrate2to3.sql`

Then in the `atlas.sum` file, move the new line that was generated with old file name to the end of the list and change _only the filename_ to match your new one. After this you must run the following command to fix the hash record:

`atlas migrate hash`

In the end you should see only two changes in the file according to git. The base hash (line 1) will be updated, and the new inserted line at the end referencing your SQL file. All hashes for previous steps should remain the same.

#### Examine the new file

Once your file is created, examine it to make sure the changes make sense.

A very common migration pattern is to create a new temporary file for an altered table, copy the data from the old file to it, then delete the old table and rename the new one to the original name.

If the SQL file is changing things you didn't anticipate or can't account for, then someone may have made changes to a table you are not aware of. Resolve these issues before proceeding.

### Update the migration GO code

The file `migrate.go` contains the code needed to fire each SQL upgrade script in the proper sequential order. It uses the PRAGMA command to find the user_version number in the database then executes the appropriate SQL scripts to update the database to the latest version.

After adding changes necessary to support the latest DB changes, you should compile and run the migrate app to ensure it works, then test and examine the resulting DB to make sure it has all the appropriate changes.

### Delete and Revert Unneeded Changes

Assuming the migration script works as intended, you'll need to undo the changes to

`igor2/internal/app/igor-server/tools.go`

`igor2/go.mod`

`igor2/go.sum`

and anything in the db-migrate folder you don't wish to keep.

### Save your upgrade to the repo

This should include:

```
db-migrate/migrations/atlas.sum
db-migrate/migrations/migrateXtoY.sql
db-migrate/test-dbs/igor.vX-test.db    # example database you are migrating FROM
db-migrate/migrate.go
```
And any other changes files deemed necessary to keep.