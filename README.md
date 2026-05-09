# Architectural Blueprint

A robust, multi-project architectural management system that allows users to design, persist, and manage system infrastructure blueprints using a modular React frontend and a high-performance Go backend.

## 🏗 Architecture

The project is built with a decoupled architecture:
- **Frontend**: React-based canvas using **React Flow** for the diagramming engine and **Framer Motion** for a premium UI/UX.
- **Backend**: Go API server utilizing the **Strategy Pattern** for database operations and logging.
- **Database**: PostgreSQL for persistent storage of projects, nodes, and connections.

## 🚀 Features

- **Multi-Project Management**: Create, list, and delete multiple architecture projects.
- **Interactive Canvas**: Drag-and-drop infrastructure blocks (API Gateways, Load Balancers, Databases, etc.).
- **Real-time Persistence**: Node positions and configurations are automatically saved to the database.
- **Transactional Safety**: Atomic operations for deleting projects and nodes to maintain data integrity.
- **Premium Design**: Modern aesthetic with glassmorphism and smooth animations.

## 🛠 Tech Stack

- **Backend**: Go (Golang)
- **Frontend**: React, Vite, TypeScript, React Flow, Framer Motion, Lucide React
- **Database**: PostgreSQL (Dockerized)
- **Networking**: Axios (Frontend), `github.com/rs/cors` (Backend)

---

## 🚦 Getting Started

### Prerequisites

- **Go**: 1.18 or higher
- **Node.js**: 16.x or higher (npm or yarn)
- **Docker**: For running the PostgreSQL instance

### 1. Database Setup

The application expects a PostgreSQL instance. The easiest way to start is using Docker:

```bash
docker run --name local-pg -e POSTGRES_PASSWORD=postgres -p 5432:5432 -d postgres
```

The application will automatically perform migrations and create the necessary tables (`projects`, `nodes`, `connections`) upon startup.

### 2. Backend Configuration

Create a `.env` file in the root directory:

```env
hostname=localhost
port=5432
POSGRES_PASSWORD=postgres
```

### 3. Running the Backend

Install dependencies and start the Go server:

```bash
go mod tidy
go run main.go
```

The API will be available at `http://localhost:8080`.

### 4. Running the Frontend

Navigate to the `app` directory, install dependencies, and start the development server:

```bash
cd app
npm install
npm run dev
```

The frontend will be available at `http://localhost:5173`.

---

## 📂 Project Structure

```text
├── api/                # Core Go API logic
│   ├── models/         # Database models and SQL schemas
│   └── ...             # Strategy implementations (DB, Logging)
├── app/                # React Frontend
│   ├── src/
│   │   ├── components/ # Reusable UI and Canvas components
│   │   └── pages/      # Landing and Canvas pages
├── api_gateway/        # Experimental Gateway logic
├── common/             # Shared Go types
└── main.go             # Application entry point
```

## 📝 License

This project is for educational purposes. Feel free to modify and extend it!