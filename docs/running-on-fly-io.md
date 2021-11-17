This is HOW-TO run your Checker instance on fly.io service free virtual machine.
First of all, you need sign up on the fly.io if you haven't already.
They'll ask you to enter credit card information to create your first app, but you can just buy some credits,
this will allow to run three minimal virtual machines, which is enough to run our Checker instance.

If you are not familiar with fly.io, please follow their guide after sign-up here:
https://fly.io/docs/hands-on/start/

When you ready to go with Checker, please fork this project.
After forking, you'll need create to a new GitHub Environment in your fork, do it project's Settings->Environments.
Let's name our test environment as `checker-staging-fly`.

Now, you need to create fly.io access token to be used in GitHub Actions pipeline.
Go to your fly.io account page, Settings->Active tokens.
Create new token, give it some reasonable name (e.g. `github-actions-checker-staging-fly`), and copy its content.
Note that token content displayed only once, and after you click `Return to list` youll not be able to retrieve it if lost.
Just create new one, and delete old lost token.

Next, please create a secret inside `checker-staging-fly` environment configuration, named `FLY_API_TOKEN`
and fill it with token, that you got on previous step.

