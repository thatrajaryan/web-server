import React, { useState, useCallback, useRef, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
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
  Panel
} from 'reactflow';
import 'reactflow/dist/style.css';
import { ChevronLeft, Share2, Play, Square, Trash2 } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { CustomNode } from '../components/Canvas/CustomNode';
import { BlockPalette, blockTypes } from '../components/Sidebar/BlockPalette';
import { apiClient } from '../api/client';

const nodeTypes = {
  custom: CustomNode,
};

export const CanvasPage = () => {
  const { projectId } = useParams();
  const navigate = useNavigate();
  const reactFlowWrapper = useRef<HTMLDivElement>(null);
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [reactFlowInstance, setReactFlowInstance] = useState<any>(null);
  const [selectedNode, setSelectedNode] = useState<Node | null>(null);

  useEffect(() => {
    const loadProject = async () => {
      try {
        const response = await apiClient.get(`/project/${projectId}/details`);
        const { nodes: savedNodes, connections: savedEdges } = response.data.data;
        
        // Map saved data to React Flow format
        setNodes(savedNodes.map((n: any) => ({
          ...n,
          type: 'custom',
          data: { 
            ...n.config, 
            label: blockTypes.find(b => b.type === n.type)?.label,
            icon: blockTypes.find(b => b.type === n.type)?.icon,
            id: n.id
          }
        })));
        
        setEdges(savedEdges.map((e: any) => ({
          id: e.id,
          source: e.from_node_id,
          target: e.to_node_id
        })));
      } catch (error) {
        console.error('Failed to load project details:', error);
      }
    };

    if (projectId) loadProject();
  }, [projectId, setNodes, setEdges]);

  const onConnect = useCallback(async (params: Connection | Edge) => {
    setEdges((eds) => addEdge(params, eds));
    try {
      await apiClient.post('/create/connection', {
        project_id: projectId,
        from_id: params.source,
        to_id: params.target
      });
    } catch (error) {
      console.error('Failed to create connection:', error);
    }
  }, [projectId, setEdges]);

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
      if (!type) return;

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

      try {
        await apiClient.post(`/create/${type.replace('_', '-')}`, {
          id: id,
          project_id: projectId,
          config: {}
        });
      } catch (error) {
        console.error(`Failed to create ${type}:`, error);
      }
    },
    [reactFlowInstance, projectId, setNodes]
  );

  return (
    <div className="canvas-container" ref={reactFlowWrapper} style={{ height: '100vh', width: '100vw' }}>
      <ReactFlowProvider>
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onConnect={onConnect}
          onInit={setReactFlowInstance}
          onDrop={onDrop}
          onDragOver={onDragOver}
          onNodeClick={(_, node) => setSelectedNode(node)}
          nodeTypes={nodeTypes}
          fitView
        >
          <Background color="#1e293b" gap={20} />
          <Controls />
          
          <Panel position="top-left" style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
            <button 
              onClick={() => navigate('/')} 
              className="btn" 
              style={{ background: 'var(--panel-bg)', backdropFilter: 'blur(12px)', display: 'flex', alignItems: 'center', gap: '8px', border: '1px solid var(--border-color)' }}
            >
              <ChevronLeft size={20} /> Back to Projects
            </button>
            <BlockPalette />
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
                  <h3>Block Details</h3>
                  <div className="input-group">
                    <label>Node ID</label>
                    <input value={selectedNode.id} readOnly />
                  </div>
                  <div className="input-group">
                    <label>Type</label>
                    <input value={selectedNode.data.type} readOnly />
                  </div>
                  <div style={{ display: 'flex', gap: '10px', marginTop: 'auto' }}>
                    <button className="btn" style={{ flex: 1 }}><Play size={16} /> Start</button>
                    <button className="btn" style={{ flex: 1, background: '#475569' }}><Square size={16} /> Stop</button>
                  </div>
                  <button className="btn" style={{ background: '#ef4444' }} onClick={() => setSelectedNode(null)}>
                    <Trash2 size={16} /> Close
                  </button>
                </motion.div>
              </Panel>
            )}
          </AnimatePresence>

          <button className="connection-btn">
            <Share2 size={20} /> Deploy Architecture
          </button>
        </ReactFlow>
      </ReactFlowProvider>
    </div>
  );
};
