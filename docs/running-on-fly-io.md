#HOW-TO
##Run your Checker instance on fly.io service free virtual machine.

First of all, you need sign up on the fly.io if you haven't already.
They'll ask you to enter credit card information to create your first app, but you can just buy some credits,
this will allow to run three minimal virtual machines, which is enough to run our Checker instance.

If you are not familiar with fly.io, please follow their guide after sign-up here:
https://fly.io/docs/hands-on/start/

When you ready to go with Checker, please fork this project.
After forking, you'll need create to a new GitHub Environment in your fork, do it in project's Settings->Environments.
Let's name our test environment as `checker-staging-fly-<your-GitHub-username(or some random string)>`.
This name needs to be unique across all fly.io applications, cause each app gets its own DNS name.
So choose it carefully, and we will use this DNS name further to access the app.

Now, you need to create fly.io access token to be used in GitHub Actions pipeline.
Go to your fly.io account page, Settings->Active tokens.
Create new token, give it some reasonable name (e.g. `github-actions-checker-staging-fly`), and copy its content.
Note that token content displayed only once, and after you click `Return to list` you'll not be able to retrieve it.
If lost, just create new one, and delete lost token.

Next, please create a secret inside `checker-staging-fly` GitHub environment configuration, named `FLY_API_TOKEN`
and fill it with token, gotten on previous step.

Edit `.github/workflows/master.yml` in your project fork, and set your personal environment name in `app`,
and `jobs.build.environment` properties.

After commit your changes into master branch, new Actions pipeline should be triggered, you can monitor it on
Actions->Environment name page.

After successful Action finish, you should get your app deployed at `https://fly.io/apps/<your-app-name>`.
Also try to ping it by http:
```
❯ curl https://<your-app-name>.fly.dev/healthcheck
Ok!
```

With default configuration, it will just check https://google.com availability once per minute and output to log.
App's log can be monitored using: `flyctl logs -a <your-app-name>`.

## Monitor Checker's log using logtail.com free plan

## Use Telegram to get alerts from Checker