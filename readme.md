# jats

jats is Just Another To-do System - a task-centric system with an optional email integration. Also includes time tracking and reporting with a rich TUI interface. _Disclaimer: I built this using Claude Code. I did not save my conversation unfortunately to include in my commits. Not sure if I will use this long term, but still throwing the code over the wall for now._

## Architecture

The architecture is simple - everything is based on the root object of a task. A task can be as simple as "update docker to latest version". I can mark that task as completed and everything is happy.

However, tasks can also be made complicated. Tasks can have comments with additional information. Tasks can be worked on over time, with time being added every time progress is made. Tasks can be created by bcc'ing a IMAP inbox and receive updates with further emails. Tasks can also have tags which can be used to sort them, include subtasks, etc. 

## Deployment

Instructions (potentially) coming soon. Depends if I decide this project is good enough to keep long term.

## Why / How

I couldn't find an appropriate system that properly combined to-do list, ticketing, and time tracking. I originally built a super simple todo cli app, but then needed to start tracking how much time was being spent on each task. I retro-fitted that in, but then I needed to start making reports based off the data. Then, I wished I had an easy way to forward emails to add them to the system. So, logicically, I decided to give up on all the work I had done previously to build something new with a server backend and rich CLI. I had many co-workers tell me the glory that is Claude Code, so I figured this was a nice project to test it with. _In case my opinion is wanted, personally I find using Claude took out all the joy I normally receive when building a project. I doubt I will continue to use it for personal projects. That being said, it did better than I expected and plan to leave it as a tool in my toolbelt as needed._
