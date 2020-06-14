gooser
======
gooser is a small GRPC user api written in golang. It provides:
* functions for creating, updating, deleting users & groups
* functions for resetting the password
* groups can have roles assigned

# settings
All settings have to be provided by environment variables:

| environment   variable         | description                                                                                                                                        | default                                |
|--------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------|----------------------------------------|
| GOOSER_ADMIN_USER              | A user with the given username will be created if it does not exist. The   user will be put in a group called "admins", having the "admin" role.   | admin                                  |
| GOOSER_CONFIRM_URL             | Base url which will be sent for confirming the user's mail address                                                                                 | http://localhost:1234/#/confirm-mail   |
| GOOSER_DEFAULT_LANGUAGE        | Default language to be used                                                                                                                        | en                                     |
| GOOSER_MAIL_FROM               | The mail address from which mails will be sent by the server                                                                                       | the value from GOOSER_SMTP_USERNAME    |
| GOOSER_MONGO_DB                | Name of the mongodb database                                                                                                                       | db                                     |
| GOOSER_MONGO_GROUPS_COLLECTION | Name of the mongodb groups collection                                                                                                              | groups                                 |
| GOOSER_MONGO_URL               | Url for the mongodb connection                                                                                                                     | mongodb://localhost:27017              |
| GOOSER_MONGO_USERS_COLLECTION  | Name of the mongodb users collection                                                                                                               | users                                  |
| GOOSER_OAUTH_URL               | Base url for oauth (will be used to query /userinfo)                                                                                               | http://localhost:4444                  |
| GOOSER_PORT                    | Port on which the server should be run                                                                                                             | 50051                                  |
| GOOSER_RESET_PASSWORD_URL      | Base url for resetting passwords                                                                                                                   | http://localhost:1234/#/reset-password |
| GOOSER_SECRET                  | Secret used for encryption. Make sure to set this variable in production!                                                                          |                                        |
| GOOSER_SITE_NAME               | Site name used in mails                                                                                                                            | gooser                                 |
| GOOSER_SMTP_HOST               | Hostname for the smtp connection. If not defined, mails will be written to stdout.                                                                 |                                        |
| GOOSER_SMTP_PASSWORD           | Password for the smtp connection                                                                                                                   |                                        |
| GOOSER_SMTP_PORT               | Port for the smtp connection                                                                                                                       | 587                                    |
| GOOSER_SMTP_USERNAME           | Username for the smtp connection                                                                                                                   |                                        |