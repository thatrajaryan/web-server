import axios from 'axios';

export const API_BASE = 'http://localhost:8080';

export const apiClient = axios.create({
  baseURL: API_BASE,
  headers: {
    'Content-Type': 'application/json',
  },
});

export interface Project {
  id: string;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface NodeData {
  id: string;
  project_id: string;
  type: string;
  config: any;
}

export interface ConnectionData {
  id: string;
  project_id: string;
  from_node_id: string;
  to_node_id: string;
}
