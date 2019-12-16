const { events, Job } = require("brigadier");

const projectOrg = "cnabio";
const projectName = "cnab-go";

const goImg = "golang:1.13";
const gopath = "/go";
const localPath = gopath + `/src/github.com/${projectOrg}/${projectName}`;

const releaseTagRegex = /^refs\/tags\/(v[0-9]+(?:\.[0-9]+)*(?:\-.+)?)$/;

// **********************************************
// Event Handlers
// **********************************************

events.on("exec", (e, p) => {
  return test(e, p).run();
})

events.on("push", (e, p) => {
  let matchStr = e.revision.ref.match(releaseTagRegex);

  if (matchStr) {
    // This is an official release with a semantically versioned tag
    let matchTokens = Array.from(matchStr);
    let version = matchTokens[1];
    return test(e, p).run()
      .then(() => {
        githubRelease(p, version).run();
      });
  }
})

events.on("check_suite:requested", runSuite);
events.on("check_suite:rerequested", runSuite);
events.on("check_run:rerequested", checkRequested);
events.on("issue_comment:created", handleIssueComment);
events.on("issue_comment:edited", handleIssueComment);

// **********************************************
// Actions
// **********************************************

function test(e, project) {
  var test = new Job("tests", goImg);

  // Add a bit of set up for golang build
  test.env = {
    DEST_PATH: localPath,
    GOPATH: gopath
  };

  test.tasks = [
    // Need to move the source into GOPATH so vendor/ works as desired.
    `mkdir -p ${localPath}`,
    `cp -a /src/* ${localPath}`,
    `cp -a /src/.git ${localPath}`,
    `cd ${localPath}`,
    "make bootstrap",
    "make build",
    "make test",
    "make lint",
  ];

  return test;
}

// Here we can add additional Check Runs, which will run in parallel and
// report their results independently to GitHub
function runSuite(e, p) {
  // For now, this is the one-stop shop running build, lint and test targets
  return runTests(e, p);
}

// runTests is a Check Run that is ran as part of a Checks Suite
function runTests(e, p) {
  console.log("Check requested");

  // Create Notification object (which is just a Job to update GH using the Checks API)
  var note = new Notification(`tests`, e, p);
  note.conclusion = "";
  note.title = "Run Tests";
  note.summary = "Running the test targets for " + e.revision.commit;
  note.text = "This test will ensure build, linting and tests all pass."

  // Send notification, then run, then send pass/fail notification
  return notificationWrap(test(e, p), note);
}

// handleIssueComment handles an issue_comment event, parsing the comment
// text and determining whether or not to trigger a corresponding action
function handleIssueComment(e, p) {
  if (e.payload) {
    payload = JSON.parse(e.payload);

    // Extract the comment body and trim whitespace
    comment = payload.body.comment.body.trim();

    // Here we determine if a comment should provoke an action
    switch(comment) {
    case "/brig run":
      return runTests(e, p);
    default:
      console.log(`No applicable action found for comment: ${comment}`);
    }
  }
}

// checkRequested is the default function invoked on a check_run:* event
//
// It determines which check is being requested (from the payload body)
// and runs this particular check, or else throws an error if the check
// is not found
function checkRequested(e, p) {
  if (e.payload) {
    payload = JSON.parse(e.payload);

    // Extract the check name
    name = payload.body.check_run.name;

    // Determine which check to run
    switch(name) {
    case "tests":
      return runTests(e, p);
    default:
      throw new Error(`No check found with name: ${name}`);
    }
  }
}

// githubRelease creates a new release on GitHub, named by the provided tag
function githubRelease(p, tag) {
  if (!p.secrets.ghToken) {
    throw new Error("Project must have 'secrets.ghToken' set");
  }

  var job = new Job("release", goImg);
  job.mountPath = localPath;
  parts = p.repo.name.split("/", 2);

  job.env = {
    "GITHUB_USER": parts[0],
    "GITHUB_REPO": parts[1],
    "GITHUB_TOKEN": p.secrets.ghToken,
  };

  job.tasks = [
    "go get github.com/aktau/github-release",
    `cd ${localPath}`,
    `last_tag=$(git describe --tags ${tag}^ --abbrev=0 --always)`,
    `github-release release \
      -t ${tag} \
      -n "${parts[1]} ${tag}" \
      -d "$(git log --no-merges --pretty=format:'- %s %H (%aN)' HEAD ^$last_tag)" \
      || echo "release ${tag} exists"`
  ];

  console.log(job.tasks);
  console.log(`release at https://github.com/${p.repo.name}/releases/tag/${tag}`);

  return job;
}


// **********************************************
// Classes/Helpers
// **********************************************

// A GitHub Check Suite notification
class Notification {
  constructor(name, e, p) {
    this.proj = p;
    this.payload = e.payload;
    this.name = name;
    this.externalID = e.buildID;
    this.detailsURL = `https://brigadecore.github.io/kashti/builds/${ e.buildID }`;
    this.title = "running check";
    this.text = "";
    this.summary = "";

    // count allows us to send the notification multiple times, with a distinct pod name
    // each time.
    this.count = 0;

    // One of: "success", "failure", "neutral", "cancelled", or "timed_out".
    this.conclusion = "neutral";
  }

  // Send a new notification, and return a Promise<result>.
  run() {
    this.count++;
    var job = new Job(`${ this.name }-notification-${ this.count }`, "brigadecore/brigade-github-check-run:v0.1.0");
    job.imageForcePull = true;
    job.env = {
      "CHECK_CONCLUSION": this.conclusion,
      "CHECK_NAME": this.name,
      "CHECK_TITLE": this.title,
      "CHECK_PAYLOAD": this.payload,
      "CHECK_SUMMARY": this.summary,
      "CHECK_TEXT": this.text,
      "CHECK_DETAILS_URL": this.detailsURL,
      "CHECK_EXTERNAL_ID": this.externalID
    };
    return job.run();
  }
}

// Helper to wrap a job execution between two notifications.
async function notificationWrap(job, note, conclusion) {
  if (conclusion == null) {
    conclusion = "success"
  }
  await note.run();
  try {
    let res = await job.run();
    const logs = await job.logs();
    note.conclusion = conclusion;
    note.summary = `Task "${ job.name }" passed`;
    note.text = "```" + res.toString() + "```\nTest Complete";
    return await note.run();
  } catch (e) {
    const logs = await job.logs();
    note.conclusion = "failure";
    note.summary = `Task "${ job.name }" failed for ${ e.buildID }`;
    note.text = "```" + logs + "```\nFailed with error: " + e.toString();
    try {
      await note.run();
    } catch (e2) {
      console.error("failed to send notification: " + e2.toString());
      console.error("original error: " + e.toString());
    }
    throw e;
  }
}
