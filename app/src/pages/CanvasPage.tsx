import React, { useState, useCallback, useRef, useEffect } from 'react';
import ReactFlow, { 
  addEdge, 
  Background, 
  Controls, 
  Panel,
  ReactFlowProvider,
  useNodesState,
  useEdgesState,
  type Node,
  type Edge,
  type Connection,
} from 'reactflow';
import 'reactflow/dist/style.css';
import { useNavigate, useParams } from 'react-router-dom';
import { ChevronLeft, Play, Square, Trash2, X, Share2 } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { BlockPalette, blockTypes } from '../components/Sidebar/BlockPalette';
import { CustomNode } from '../components/Canvas/CustomNode';
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
  const [selectedEdge, setSelectedEdge] = useState<Edge | null>(null);

  useEffect(() => {
    const loadProject = async () => {
      if (!projectId) return;
      try {
        const response = await apiClient.get(`/project/${projectId}/details`);
        const { nodes: savedNodes, connections: savedConns } = response.data.data;
        
        setNodes(savedNodes.map((n: any) => ({
          id: n.id,
          type: 'custom',
          position: n.config?.position || { x: 0, y: 0 },
          data: { 
            ...n.config, 
            label: blockTypes.find(b => b.type === n.type)?.label,
            icon: blockTypes.find(b => b.type === n.type)?.icon,
            id: n.id,
            type: n.type
          }
        })));
        
        setEdges(savedConns.map((c: any) => ({
          id: c.id,
          source: c.from_node_id,
          target: c.to_node_id,
        })));
      } catch (error) {
        console.error('Failed to load project:', error);
      }
    };
    loadProject();
  }, [projectId, setNodes, setEdges]);

  const onConnect = useCallback(async (params: Connection) => {
    setEdges((eds) => addEdge(params, eds));
    try {
      await apiClient.post('/create/connection', {
        project_id: projectId,
        from_id: params.source,
        to_id: params.target
      });
    } catch (error) {
      console.error('Failed to save connection:', error);
    }
  }, [projectId, setEdges]);

  const onEdgesDelete = useCallback(async (deletedEdges: Edge[]) => {
    for (const edge of deletedEdges) {
      try {
        await apiClient.delete(`/connection/delete?source=${edge.source}&target=${edge.target}`);
      } catch (error) {
        console.error('Failed to delete connection:', error);
      }
    }
  }, []);

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
          config: { position }
        });
      } catch (error) {
        console.error(`Failed to create ${type}:`, error);
      }
    },
    [reactFlowInstance, projectId, setNodes]
  );

  const onNodeDragStop = useCallback(async (_: any, node: Node) => {
    try {
      await apiClient.put(`/block/update?id=${node.id}`, {
        config: { ...node.data, position: node.position }
      });
    } catch (error) {
      console.error('Failed to update node position:', error);
    }
  }, []);

  const onEdgeClick = useCallback((_: any, edge: Edge) => {
    setSelectedNode(null);
    setSelectedEdge(edge);
  }, []);

  const onNodeClick = useCallback((_: any, node: Node) => {
    setSelectedEdge(null);
    setSelectedNode(node);
  }, []);

  const handleDeleteNode = useCallback(async () => {
    if (!selectedNode) return;
    if (!window.confirm(`Delete ${selectedNode.data.type}?`)) return;
    try {
      await apiClient.delete(`/block/delete?id=${selectedNode.id}`);
      setNodes((nds) => nds.filter((n) => n.id !== selectedNode.id));
      setEdges((eds) => eds.filter((e) => e.source !== selectedNode.id && e.target !== selectedNode.id));
      setSelectedNode(null);
    } catch (error) {
      console.error('Failed to delete node:', error);
    }
  }, [selectedNode, setNodes, setEdges]);

  const handleDeleteEdge = useCallback(async () => {
    if (!selectedEdge) return;
    if (!window.confirm('Delete connection?')) return;
    try {
      await apiClient.delete(`/connection/delete?source=${selectedEdge.source}&target=${selectedEdge.target}`);
      setEdges((eds) => eds.filter((e) => e.id !== selectedEdge.id));
      setSelectedEdge(null);
    } catch (error) {
      console.error('Failed to delete edge:', error);
    }
  }, [selectedEdge, setEdges]);

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
          onNodeClick={onNodeClick}
          onEdgeClick={onEdgeClick}
          onNodeDragStop={onNodeDragStop}
          onEdgesDelete={onEdgesDelete}
          nodeTypes={nodeTypes}
          fitView
        >
          <Background color="#1e293b" gap={20} />
          <Controls />
          
          <Panel position="top-left" style={{ margin: '20px' }}>
            <div style={{ 
              display: 'flex', flexDirection: 'column', gap: '16px', background: 'var(--panel-bg)',
              backdropFilter: 'blur(16px)', padding: '20px', borderRadius: '24px',
              border: '1px solid var(--border-color)', boxShadow: '0 20px 40px rgba(0,0,0,0.4)', width: '260px'
            }}>
              <button 
                onClick={() => navigate('/')} className="btn" 
                style={{ width: '100%', display: 'flex', alignItems: 'center', gap: '10px', background: 'rgba(255,255,255,0.05)', border: '1px solid rgba(255,255,255,0.1)', padding: '12px' }}
              >
                <ChevronLeft size={18} /> Back to Projects
              </button>
              <div style={{ height: '1px', background: 'var(--border-color)', margin: '4px 0' }} />
              <BlockPalette />
            </div>
          </Panel>

          <AnimatePresence>
            {(selectedNode || selectedEdge) && (
              <Panel position="top-right" style={{ margin: '20px' }}>
                <motion.div 
                  initial={{ x: 320, opacity: 0 }} animate={{ x: 0, opacity: 1 }} exit={{ x: 320, opacity: 0 }}
                  style={{
                    background: 'var(--panel-bg)', backdropFilter: 'blur(16px)', padding: '24px', borderRadius: '24px',
                    border: '1px solid var(--border-color)', boxShadow: '0 20px 40px rgba(0,0,0,0.4)', width: '300px', color: '#fff'
                  }}
                >
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
                    <h3 style={{ margin: 0, fontSize: '1.2rem' }}>{selectedNode ? 'Block Config' : 'Link Config'}</h3>
                    <button onClick={() => { setSelectedNode(null); setSelectedEdge(null); }} style={{ background: 'transparent', border: 'none', color: 'var(--text-secondary)', cursor: 'pointer' }}>
                      <X size={20} />
                    </button>
                  </div>
                  
                  {selectedNode ? (
                    <>
                      <div className="input-group" style={{ marginBottom: '16px' }}>
                        <label style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', textTransform: 'uppercase' }}>ID</label>
                        <input value={selectedNode.id} readOnly style={{ background: 'rgba(0,0,0,0.2)', border: '1px solid rgba(255,255,255,0.05)', color: 'rgba(255,255,255,0.5)' }} />
                      </div>
                      <div className="input-group" style={{ marginBottom: '24px' }}>
                        <label style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', textTransform: 'uppercase' }}>Type</label>
                        <div style={{ background: 'rgba(59, 130, 246, 0.1)', color: '#3b82f6', padding: '10px', borderRadius: '12px', textAlign: 'center', fontWeight: 600 }}>
                          {selectedNode.data.type?.replace('_', ' ')}
                        </div>
                      </div>
                      <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                        <button className="btn" style={{ width: '100%', background: '#3b82f6' }}><Play size={16} /> Start Block</button>
                        <button className="btn" style={{ width: '100%', background: 'rgba(255, 255, 255, 0.05)' }}><Square size={16} /> Stop Block</button>
                        <div style={{ height: '1px', background: 'var(--border-color)', margin: '8px 0' }} />
                        <button onClick={handleDeleteNode} className="btn" style={{ width: '100%', background: '#ef4444', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px' }}>
                          <Trash2 size={16} /> Delete Node
                        </button>
                      </div>
                    </>
                  ) : selectedEdge && (
                    <>
                      <div className="input-group" style={{ marginBottom: '16px' }}>
                        <label style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', textTransform: 'uppercase' }}>From</label>
                        <div style={{ fontSize: '0.8rem', padding: '8px', background: 'rgba(0,0,0,0.2)', borderRadius: '8px', fontFamily: 'monospace' }}>{selectedEdge.source}</div>
                      </div>
                      <div className="input-group" style={{ marginBottom: '24px' }}>
                        <label style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', textTransform: 'uppercase' }}>To</label>
                        <div style={{ fontSize: '0.8rem', padding: '8px', background: 'rgba(0,0,0,0.2)', borderRadius: '8px', fontFamily: 'monospace' }}>{selectedEdge.target}</div>
                      </div>
                      <button onClick={handleDeleteEdge} className="btn" style={{ width: '100%', background: '#ef4444', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px' }}>
                        <Trash2 size={16} /> Delete Connection
                      </button>
                    </>
                  )}
                </motion.div>
              </Panel>
            )}
          </AnimatePresence>

          <button className="connection-btn" style={{
            position: 'fixed', bottom: '40px', left: '50%', transform: 'translateX(-50%)',
            background: 'linear-gradient(135deg, #3b82f6, #8b5cf6)', color: '#fff', padding: '16px 32px', borderRadius: '100px',
            border: 'none', display: 'flex', alignItems: 'center', gap: '12px', fontSize: '1rem', fontWeight: 600,
            cursor: 'pointer', boxShadow: '0 10px 40px rgba(59, 130, 246, 0.4)', zIndex: 10
          }}>
            <Share2 size={20} /> Deploy Architecture
          </button>
        </ReactFlow>
      </ReactFlowProvider>
    </div>
  );
};
