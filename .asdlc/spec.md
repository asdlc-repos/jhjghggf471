# Overview

A web-based task management application that enables users to create, organize, and track personal todo items with authentication, categorization, and deadline tracking capabilities.

# Personas

- **Sarah** - Busy Professional: Needs to organize work and personal tasks across multiple projects with clear deadlines
- **Mike** - Student: Requires a simple system to track assignments, study sessions, and extracurricular activities by category
- **Admin** - System Administrator: Manages user accounts and monitors system health

# Capabilities

## User Authentication

- The system SHALL provide user registration with email and password
- The system SHALL validate email format during registration
- The system SHALL enforce password minimum length of 8 characters
- WHEN a user attempts login, the system SHALL verify credentials against stored records
- IF login credentials are invalid, THEN the system SHALL display an error message and deny access
- IF a user fails authentication three consecutive times, THEN the system SHALL temporarily lock the account for 15 minutes
- WHEN a user successfully authenticates, the system SHALL create a session valid for 24 hours
- The system SHALL provide a logout function that terminates the active session
- The system SHALL provide password reset functionality via email verification

## Task Management

- WHILE authenticated, a user SHALL be able to create new tasks with title and description
- WHILE authenticated, a user SHALL be able to view all their tasks
- WHILE authenticated, a user SHALL be able to edit existing task details
- WHILE authenticated, a user SHALL be able to delete tasks
- WHEN a user creates a task, the system SHALL assign a unique identifier to that task
- The system SHALL store task creation timestamp for each task
- WHEN a user marks a task as complete, the system SHALL update the task status and record completion timestamp
- The system SHALL allow users to mark completed tasks as incomplete
- The system SHALL display tasks in chronological order by creation date by default

## Categories

- The system SHALL allow users to create custom categories with unique names
- The system SHALL allow users to assign one category to each task
- The system SHALL allow users to view tasks filtered by category
- The system SHALL allow users to rename existing categories
- WHEN a user deletes a category, the system SHALL unassign that category from all associated tasks
- The system SHALL prevent creation of duplicate category names for the same user
- The system SHALL allow tasks to exist without an assigned category

## Due Dates

- The system SHALL allow users to assign a due date to any task
- The system SHALL accept due dates in YYYY-MM-DD format
- WHEN viewing tasks, the system SHALL display days remaining until due date
- The system SHALL allow users to view tasks filtered by due date range
- IF a task due date has passed and the task is incomplete, THEN the system SHALL visually flag the task as overdue
- The system SHALL allow tasks to exist without a due date
- The system SHALL allow users to modify or remove due dates from existing tasks

## Data Security

- The system SHALL store passwords using bcrypt hashing with minimum cost factor of 10
- The system SHALL enforce user data isolation ensuring users can only access their own tasks and categories
- The system SHALL use HTTPS for all client-server communication in production
- The system SHALL validate and sanitize all user input to prevent injection attacks
- IF an unauthorized access attempt is detected, THEN the system SHALL deny the request and log the incident

## Performance

- WHEN a user performs any CRUD operation, the system SHALL respond within 500 milliseconds under normal load
- The system SHALL support up to 100 concurrent authenticated users
- The system SHALL handle up to 10,000 tasks per user without performance degradation

## User Interface

- The system SHALL provide a responsive interface that adapts to desktop and mobile screen sizes
- WHILE viewing the task list, the system SHALL display task title, category, due date, and completion status
- The system SHALL provide visual feedback within 200 milliseconds for all user actions
- IF a user action fails, THEN the system SHALL display a clear error message describing the issue
- The system SHALL provide keyboard navigation for all primary functions