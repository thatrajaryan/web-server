import React, { useState, useCallback, useRef } from 'react';
import ReactFlow, {
  addEdge,
  Background,
  Controls,
  type Connection,
  type Edge,
  type Node,
  ReactFlowProvider,
  useNodesState,
  useEdgesState,
  Panel,
  Handle,
  Position
} from 'reactflow';
import 'reactflow/dist/style.css';
import {
  Zap,
  Cpu,
  Database,
  Share2,
  Layers,
  ShieldAlert,
  Server as ServerIcon,
  Globe,
  Trash2,
  Play,
  Square
} from 'lucide-react';
import axios from 'axios';
import { motion, AnimatePresence } from 'framer-motion';

const API_BASE = 'http://localhost:8080';

const blockTypes = [
  { type: 'api_gateway', label: 'API Gateway', icon: <Zap size={18} /> },
  { type: 'load_balancer', label: 'Load Balancer', icon: <Layers size={18} /> },
  { type: 'code', label: 'Code Block', icon: <Cpu size={18} /> },
  { type: 'kafka', label: 'Kafka', icon: <Share2 size={18} /> },
  { type: 'database', label: 'Database', icon: <Database size={18} /> },
  { type: 'rate_limiter', label: 'Rate Limiter', icon: <ShieldAlert size={18} /> },
  { type: 'server', label: 'Server', icon: <ServerIcon size={18} /> },
  { type: 'cdn', label: 'CDN', icon: <Globe size={18} /> },
];

const CustomNode = ({ data, selected }: any) => {
  return (
    <div className={`node-custom ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Top} style={{ background: '#3b82f6' }} />
      <div className="node-header">
        {data.icon}
        <span>{data.label}</span>
      </div>
      <div className="node-id">{data.id}</div>
      <Handle type="source" position={Position.Bottom} style={{ background: '#3b82f6' }} />
    </div>
  );
};

const nodeTypes = {
  custom: CustomNode,
};

const App = () => {
  const reactFlowWrapper = useRef<HTMLDivElement>(null);
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [reactFlowInstance, setReactFlowInstance] = useState<any>(null);
  const [selectedNode, setSelectedNode] = useState<Node | null>(null);

  const onConnect = useCallback(async (params: Connection | Edge) => {
    setEdges((eds) => addEdge(params, eds));

    try {
      await axios.post(`${API_BASE}/create/connection`, {
        from_id: params.source,
        to_id: params.target
      });
      console.log('Connection created in backend');
    } catch (error) {
      console.error('Failed to create connection:', error);
    }
  }, [setEdges]);

  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }, []);

  const onDrop = useCallback(
    async (event: React.DragEvent) => {
      event.preventDefault();

      if (!reactFlowWrapper.current || !reactFlowInstance) return;

      const reactFlowBounds = reactFlowWrapper.current.getBoundingClientRect();
      const type = event.dataTransfer.getData('application/reactflow');

      if (typeof type === 'undefined' || !type) {
        return;
      }

      const position = reactFlowInstance.project({
        x: event.clientX - reactFlowBounds.left,
        y: event.clientY - reactFlowBounds.top,
      });

      const id = `${type}_${Math.random().toString(36).substr(2, 9)}`;
      const blockInfo = blockTypes.find(b => b.type === type);

      const newNode: Node = {
        id,
        type: 'custom',
        position,
        data: { label: blockInfo?.label, type, id, icon: blockInfo?.icon },
      };

      setNodes((nds) => nds.concat(newNode));

      // Call backend to create block
      try {
        await axios.post(`${API_BASE}/create/${type.replace('_', '-')}`, {
          id: id,
          config: {}
        });
        console.log(`${type} created in backend`);
      } catch (error) {
        console.error(`Failed to create ${type}:`, error);
      }
    },
    [reactFlowInstance, setNodes]
  );

  const onNodeClick = (_: any, node: Node) => {
    setSelectedNode(node);
  };

  const deleteNode = async () => {
    if (!selectedNode) return;

    try {
      await axios.delete(`${API_BASE}/block/delete?id=${selectedNode.id}`);
      setNodes((nds) => nds.filter((n) => n.id !== selectedNode.id));
      setEdges((eds) => eds.filter((e) => e.source !== selectedNode.id && e.target !== selectedNode.id));
      setSelectedNode(null);
    } catch (error) {
      console.error('Failed to delete node:', error);
    }
  };

  return (
    <div className="canvas-container" ref={reactFlowWrapper}>
      <ReactFlowProvider>
        <div style={{ height: '100%', width: '100%' }}>
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onInit={setReactFlowInstance}
            onDrop={onDrop}
            onDragOver={onDragOver}
            onNodeClick={onNodeClick}
            nodeTypes={nodeTypes}
            fitView
          >
            <Background color="#1e293b" gap={20} />
            <Controls />

            <Panel position="top-left" className="sidebar">
              <h2>Architectural Blocks</h2>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
                {blockTypes.map((block) => (
                  <div
                    key={block.type}
                    className="block-item"
                    onDragStart={(event) => {
                      event.dataTransfer.setData('application/reactflow', block.type);
                      event.dataTransfer.effectAllowed = 'move';
                    }}
                    draggable
                  >
                    {block.icon}
                    <span>{block.label}</span>
                  </div>
                ))}
              </div>
            </Panel>

            <AnimatePresence>
              {selectedNode && (
                <Panel position="top-right">
                  <motion.div
                    initial={{ x: 300, opacity: 0 }}
                    animate={{ x: 0, opacity: 1 }}
                    exit={{ x: 300, opacity: 0 }}
                    className="details-panel"
                  >
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <h3>Block Details</h3>
                      <button
                        onClick={() => setSelectedNode(null)}
                        style={{ background: 'transparent', border: 'none', color: '#94a3b8', cursor: 'pointer' }}
                      >
                        ✕
                      </button>
                    </div>

                    <div className="input-group">
                      <label>Node ID</label>
                      <input value={selectedNode.id} readOnly />
                    </div>

                    <div className="input-group">
                      <label>Type</label>
                      <input value={selectedNode.data.type} readOnly />
                    </div>

                    <div style={{ display: 'flex', gap: '10px', marginTop: 'auto' }}>
                      <button className="btn" style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px' }}>
                        <Play size={16} /> Start
                      </button>
                      <button className="btn" style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px', background: '#475569' }}>
                        <Square size={16} /> Stop
                      </button>
                    </div>

                    <button
                      className="btn"
                      style={{ background: '#ef4444', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px' }}
                      onClick={deleteNode}
                    >
                      <Trash2 size={16} /> Delete Block
                    </button>
                  </motion.div>
                </Panel>
              )}
            </AnimatePresence>

            <button className="connection-btn">
              <Share2 size={20} />
              Deploy Architecture
            </button>
          </ReactFlow>
        </div>
      </ReactFlowProvider>
    </div>
  );
};

export default App;
