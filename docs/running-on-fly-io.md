# HOW-TO run and monitor Checker instance using free IAaS services

## Run your Checker instance on fly.io service free virtual machine.

First of all, you need sign up on the fly.io if you haven't already.
They'll ask you to enter credit card information to create your first app, but you can just buy some credits instead,
this will allow you to run three minimal virtual machines, which is enough to run our Checker instance.

If you are not familiar with fly.io, please follow their guide after sign-up here:
https://fly.io/docs/hands-on/start/

When you are ready to start with Checker deployment, please fork this project.
After, you'll need to create a new GitHub Environment in your fork, do it in project's Settings->Environments.
Let's name our test environment as `checker-staging-fly-<your-GitHub-username(or some random string)>`.
This name needs to be unique across all fly.io applications, cause each app gets its own DNS name.
So choose it carefully, we will use this DNS name further to access the app.

Now, you need to create fly.io access token to be used in GitHub Actions pipeline.
Go to your fly.io account page, Settings->Active tokens.
Create new token, give it some reasonable name (e.g. `github-actions-checker-staging-fly`), and copy its content.
Note that token content displayed only once, and after you click `Return to list` you'll not be able to retrieve it.
If lost, just create new one, and delete lost token.

Next, please create a secret inside `checker-staging-fly` GitHub environment configuration, named `FLY_API_TOKEN`
and fill it with token, gotten on previous step.
Also, you need to create `CONFIG` secret containing relative path in repo to config file to be used in this env.
Let's start with 'docs/examples/google.yaml'.

Edit `.github/workflows/master.yml` in your project fork, and set your personal environment name in `name` property.

After commit your changes into master branch, new Actions pipeline should be triggered, you can monitor it on
Actions page.

After successful pipeline finish, you should get your app deployed at `https://fly.io/apps/<your-app-name>`.
Let's try to ping it by http:
```
❯ curl https://<your-app-name>.fly.dev/healthcheck
Ok!
```

With default configuration it will just check https://google.com availability once per minute and output to log.
App's log can be monitored using: `flyctl logs -a <your-app-name>`.

## Monitor Checker's log using logtail.com free plan

Signup on logtail.com service, and create new log source with type `Vector`.
This will give you `Source token`. 

Next, fork or just clone https://github.com/superfly/fly-log-shipper repository, edit its fly.toml file, adding
```toml
[env]
  LOGTAIL_TOKEN = '<logtail.com source token>'
  ACCESS_TOKEN  = '<your fly token>'
  SUBJECT       = 'logs.<your-checker-app-name>.*' 
```
Change `app` property to some convenient name e.g. `fly-log-shipper`.

Create new app and deploy log shipping service with `fly apps create fly-log-shipper && fly deploy`.
Checker's log should be now seen in logtail's Live Tail page. 

## Monitor Checker's health with betteruptime.com free plan

Signup on betteruptime.com service, and create new Monitor.
`URL to monitor` shoud be set to Checker's healthcheck url we used before (`https://<your-app-name>.fly.dev/healthcheck`).

Set other parameters to some conveniet values, for example alerts by email if not accesible, or create some on-call schedule,
follow https://docs.betteruptime.com.

Optionally you can create reverse check, using checker's `http` probe, create new HeartBeat,
and new Check with HeartBeat's url, e.g.:
```yaml
---
...
projects:
...
    - name: BetterUptime active ping
      healthchecks:
        - name: Checker Running
          parameters:
            check_period: 3600s
          checks:
            - type: http
              host: https://betteruptime.com/api/v1/heartbeat/ohquoi0Uong2Chai2AhT3ohN
              code:
                - 200
              timeout: 3s
```
to ping betteruptime.com hourly, and set up betteruptime's alert if not getting this heartbeat.

## Use Telegram to get alerts from Checker

Now, here interesting part.
Look into https://core.telegram.org/bots, or some online pages how-to get your Telegram bot registered,
and get your user ID (or create chat, invite this bot, and get chat's ID).
Basically you need two things: bot token and chat ID.
Look into [https://github.com/imcitius/checker/blob/master/docs/examples/bigconfig.yaml](docs/examples/bigconfig.yaml) to find example configuration.

Tricky thing here is how to pass bot token not pushing it into repository.
Let's create new Fly App secret, call it ALERTS_TELEGRAM_TOKEN_<alert name> with the token.
```bash
flyctl secrets set CHECKER_ALERTS_TELEGRAM_TOKEN_tg_staging="<bot token>"
flyctl secrets set CHECKER_ALERTS_TELEGRAM_CRIT_tg_staging="<critical channel id>"
flyctl secrets set CHECKER_ALERTS_TELEGRAM_NONCRIT_tg_staging="<non-critical channel id>"
```
Basically crit/non-crit channels might be the same.

Set ENV var name in telegram bot config:
```yaml
---
defaults:
  http_port: '80'
  parameters:
    check_period: 5m
    report_period: 1h
    min_health: 1
    allow_fails: 0
    noncrit_alert: tg_staging
    crit_alert: tg_staging
    command_channel: tg_staging
    mode: loud

alerts:
  - name: tg_staging
    type: telegram
    bot_token: env:CHECKER_ALERTS_TELEGRAM_TOKEN_tg_staging
    critical_channel: env:CHECKER_ALERTS_TELEGRAM_CRIT_tg_staging
    noncritical_channel: env:CHECKER_ALERTS_TELEGRAM_NONCRIT_tg_staging
...
```

To deploy your custom config, please commit it to repository, and set GitHub Secret CONFIG with relative path to config file
in repo (e.g. 'configs/custom-config.yaml').

And deploy with new config.
