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
import { ChevronLeft, Trash2, X, Share2, Save, Settings, Loader2, AlertCircle, Cpu, Upload, Download, Package, Rocket } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { BlockPalette, blockTypes } from '../components/Sidebar/BlockPalette';
import { CustomNode } from '../components/Canvas/CustomNode';
import { apiClient, API_BASE } from '../api/client';

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
  const [configSchema, setConfigSchema] = useState<any>(null);
  const [isLoadingConfig, setIsLoadingConfig] = useState(false);
  const [configError, setConfigError] = useState<string | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [isSavingProject, setIsSavingProject] = useState(false);
  const [saveStatus, setSaveStatus] = useState<{ type: 'success' | 'error', message: string } | null>(null);
  const [isGeneratingHelm, setIsGeneratingHelm] = useState(false);
  const [newCustomKey, setNewCustomKey] = useState('');
  const [newCustomValue, setNewCustomValue] = useState('');

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
            id: n.id,
            type: n.type
          }
        })));

        setEdges(savedConns.map((c: any) => ({
          id: c.id,
          source: c.from_node_id,
          target: c.to_node_id,
          data: { hook_code: c.hook_code || '' }
        })));
      } catch (error) {
        console.error('Failed to load project:', error);
      }
    };
    loadProject();
  }, [projectId, setNodes, setEdges]);

  // Handle Dual-Call Configuration Loading
  useEffect(() => {
    const fetchNodeConfigAndSchema = async () => {
      if (!selectedNode) {
        setConfigSchema(null);
        setConfigError(null);
        return;
      }

      setIsLoadingConfig(true);
      setConfigError(null);

      try {
        // 1. Fetch Structure (YAML from Backend)
        try {
          const schemaRes = await apiClient.get(`/config/${selectedNode.data.type}`);
          setConfigSchema(schemaRes.data.data);
        } catch (err) {
          console.error('Failed to fetch schema:', err);
          setConfigError('Could not load configuration schema from backend.');
        }

        // 2. Fetch Values (Database State)
        try {
          const valuesRes = await apiClient.get(`/block/details/${selectedNode.id}`);
          const freshConfig = valuesRes.data.data.config;
          const updatedNode = {
            ...selectedNode,
            data: {
              ...selectedNode.data,
              ...freshConfig
            }
          };
          setSelectedNode(updatedNode);
          setNodes((nds) => nds.map((n) => n.id === selectedNode.id ? updatedNode : n));
        } catch (err) {
          console.error('Failed to fetch DB values:', err);
          // Non-fatal error, we still have the schema and local state
        }
      } finally {
        setIsLoadingConfig(false);
      }
    };

    fetchNodeConfigAndSchema();
  }, [selectedNode?.id]);

  const onConnect = useCallback(async (params: Connection) => {
    setEdges((eds) => addEdge({ ...params, data: { hook_code: '' } }, eds));
    try {
      await apiClient.post('/create/connection', {
        project_id: projectId,
        from_id: params.source,
        to_id: params.target,
        hook_code: ''
      });
    } catch (error) {
      console.error('Failed to create connection:', error);
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
        data: { label: blockInfo?.label, type, id },
      };

      setNodes((nds) => nds.concat(newNode));
      try {
        await apiClient.post(`/create/${type}`, {
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

  const handleUpdateConfig = (field: string, value: any) => {
    if (!selectedNode) return;
    const updatedNode = {
      ...selectedNode,
      data: {
        ...selectedNode.data,
        [field]: value
      }
    };
    setSelectedNode(updatedNode);
    setNodes((nds) => nds.map((n) => n.id === selectedNode.id ? updatedNode : n));
  };

  const saveNodeConfig = async () => {
    if (!selectedNode) return;
    setIsSaving(true);
    try {
      await apiClient.put(`/block/update?id=${selectedNode.id}`, {
        config: { ...selectedNode.data, position: selectedNode.position }
      });
      setSaveStatus({ type: 'success', message: 'Node configuration saved!' });
      setTimeout(() => setSaveStatus(null), 3000);
    } catch (error) {
      console.error('Failed to save node config:', error);
      setSaveStatus({ type: 'error', message: 'Failed to save node config.' });
    } finally {
      setIsSaving(false);
    }
  };

  const handleSaveProject = async () => {
    if (!projectId) return;
    setIsSavingProject(true);
    setSaveStatus(null);
    try {
      const payload = {
        project_id: projectId,
        nodes: nodes.map(n => ({
          id: n.id,
          project_id: projectId,
          type: n.data.type,
          config: { ...n.data, position: n.position }
        })),
        connections: edges.map(e => ({
          project_id: projectId,
          from_node_id: e.source,
          to_node_id: e.target,
          hook_code: e.data?.hook_code || ''
        }))
      };

      await apiClient.post('/project/save', payload);
      setSaveStatus({ type: 'success', message: 'Project saved successfully!' });
      setTimeout(() => setSaveStatus(null), 3000);
    } catch (error) {
      console.error('Failed to save project:', error);
      setSaveStatus({ type: 'error', message: 'Failed to save project. Please try again.' });
    } finally {
      setIsSavingProject(false);
    }
  };

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

  const renderDynamicFields = () => {
    if (!configSchema || !selectedNode) return null;

    const sections: Record<string, any[]> = { "General": [] };
    configSchema.fields.forEach((field: any) => {
      const sectionName = field.section || "General";
      if (!sections[sectionName]) sections[sectionName] = [];
      sections[sectionName].push(field);
    });

    return Object.entries(sections).map(([sectionName, fields]) => (
      <div key={sectionName} style={{ marginBottom: '24px' }}>
        {sectionName !== "General" && (
          <p style={{ fontSize: '0.8rem', fontWeight: 600, color: '#3b82f6', marginBottom: '12px', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
            {sectionName}
          </p>
        )}
        <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
          {fields.map((field) => (
            <div key={field.name} className="input-group">
              <label>{field.label}</label>
              {field.type === 'select' ? (
                <select
                  value={selectedNode.data[field.name] || ''}
                  onChange={(e) => handleUpdateConfig(field.name, e.target.value)}
                  style={{ width: '100%', background: '#0f172a', border: '1px solid var(--border-color)', color: '#fff', padding: '8px', borderRadius: '8px' }}
                >
                  <option value="" disabled>Select {field.label}...</option>
                  {field.options.map((opt: any, index: number) => {
                    const value = typeof opt === 'string' ? opt : opt.value;
                    const label = typeof opt === 'string' ? opt : opt.label;
                    return <option key={index} value={value}>{label}</option>
                  })}
                </select>
              ) : field.type === 'boolean' ? (
                <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                  <input
                    type="checkbox"
                    checked={selectedNode.data[field.name] ?? false}
                    onChange={(e) => handleUpdateConfig(field.name, e.target.checked)}
                  />
                  <span style={{ fontSize: '0.9rem' }}>Enabled</span>
                </div>
              ) : (
                <input
                  type={field.type}
                  value={selectedNode.data[field.name] ?? ''}
                  placeholder={`Enter ${field.label.toLowerCase()}...`}
                  onChange={(e) => handleUpdateConfig(field.name, field.type === 'number' ? (e.target.value === '' ? '' : Number(e.target.value)) : e.target.value)}
                />
              )}
            </div>
          ))}
        </div>
      </div>
    ));
  };

  const renderCustomFields = () => {
    if (!selectedNode) return null;

    const reservedKeys = ['id', 'type', 'label', 'position', 'positionAbsolute', 'width', 'height', 'selected', 'dragging'];
    const schemaKeys = configSchema ? configSchema.fields.map((f: any) => f.name) : [];
    const customKeys = Object.keys(selectedNode.data || {}).filter(k => !reservedKeys.includes(k) && !schemaKeys.includes(k));

    return (
      <div style={{ marginTop: '24px' }}>
        <p style={{ fontSize: '0.8rem', fontWeight: 600, color: '#3b82f6', marginBottom: '12px', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
          Custom Configuration
        </p>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
          {customKeys.map(key => (
            <div key={key} className="input-group" style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
              <input 
                type="text" 
                value={key} 
                disabled 
                style={{ width: '40%', opacity: 0.7, background: 'transparent', border: '1px solid var(--border-color)', color: '#fff', padding: '8px', borderRadius: '8px', fontSize: '0.85rem' }}
              />
              <input 
                type="text" 
                value={selectedNode.data[key] ?? ''} 
                onChange={(e) => handleUpdateConfig(key, e.target.value)}
                style={{ width: '50%', background: '#0f172a', border: '1px solid var(--border-color)', color: '#fff', padding: '8px', borderRadius: '8px', fontSize: '0.85rem' }}
              />
              <button 
                onClick={() => {
                  const newData = { ...selectedNode.data };
                  delete newData[key];
                  const updatedNode = { ...selectedNode, data: newData };
                  setSelectedNode(updatedNode);
                  setNodes((nds) => nds.map((n) => n.id === selectedNode.id ? updatedNode : n));
                }}
                style={{ width: '10%', background: 'transparent', border: 'none', color: '#ef4444', cursor: 'pointer', display: 'flex', justifyContent: 'center' }}
              >
                <X size={16} />
              </button>
            </div>
          ))}
          
          <div className="input-group" style={{ display: 'flex', gap: '8px', alignItems: 'center', marginTop: '8px' }}>
            <input 
              type="text" 
              placeholder="New Key" 
              value={newCustomKey}
              onChange={(e) => setNewCustomKey(e.target.value)}
              style={{ width: '40%', background: '#0f172a', border: '1px solid var(--border-color)', color: '#fff', padding: '8px', borderRadius: '8px', fontSize: '0.85rem' }}
            />
            <input 
              type="text" 
              placeholder="Value" 
              value={newCustomValue}
              onChange={(e) => setNewCustomValue(e.target.value)}
              style={{ width: '40%', background: '#0f172a', border: '1px solid var(--border-color)', color: '#fff', padding: '8px', borderRadius: '8px', fontSize: '0.85rem' }}
            />
            <button 
              onClick={() => {
                if (newCustomKey && newCustomKey.trim() !== '') {
                  handleUpdateConfig(newCustomKey.trim(), newCustomValue);
                  setNewCustomKey('');
                  setNewCustomValue('');
                }
              }}
              style={{ width: '20%', background: 'rgba(59, 130, 246, 0.1)', border: '1px solid #3b82f6', color: '#3b82f6', cursor: 'pointer', padding: '8px', borderRadius: '8px', fontSize: '0.85rem', fontWeight: 600 }}
            >
              Add
            </button>
          </div>
        </div>
      </div>
    );
  };

  const handleGenerateHelm = async () => {
    if (!projectId) return;
    try {
      setIsGeneratingHelm(true);
      setSaveStatus(null);
      
      const downloadUrl = `${API_BASE}/project/generate-helm?project_id=${projectId}`;
      
      const link = document.createElement('a');
      link.href = downloadUrl;
      link.setAttribute('target', '_blank');
      document.body.appendChild(link);
      link.click();
      
      setTimeout(() => {
        if (document.body.contains(link)) {
          document.body.removeChild(link);
        }
      }, 100);

      setSaveStatus({ type: 'success', message: 'Helm Chart download started!' });
      setTimeout(() => setSaveStatus(null), 3000);
    } catch (error) {
      console.error('Helm generation failed:', error);
      setSaveStatus({ type: 'error', message: 'Failed to initiate Helm Chart download.' });
    } finally {
      setTimeout(() => setIsGeneratingHelm(false), 2000);
    }
  };

  const handleUploadNodeConfig = async (event: React.ChangeEvent<HTMLInputElement>) => {
    if (!selectedNode) return;
    const file = event.target.files?.[0];
    if (!file) return;

    const formData = new FormData();
    formData.append('config', file);

    try {
      setIsSaving(true);
      const response = await apiClient.post(`/block/upload-config?id=${selectedNode.id}`, formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      });
      const freshConfig = response.data.data;
      const updatedNode = {
        ...selectedNode,
        data: { ...selectedNode.data, ...freshConfig }
      };
      setSelectedNode(updatedNode);
      setNodes((nds) => nds.map((n) => n.id === selectedNode.id ? updatedNode : n));
      setSaveStatus({ type: 'success', message: 'Node configuration updated via YAML!' });
      setTimeout(() => setSaveStatus(null), 3000);
    } catch (error) {
      console.error('Node config upload failed:', error);
      setSaveStatus({ type: 'error', message: 'Failed to upload node YAML.' });
    } finally {
      setIsSaving(false);
    }
  };

  const handleUploadConfig = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    const formData = new FormData();
    formData.append('config', file);

    try {
      setIsSavingProject(true);
      const response = await apiClient.post('/project/upload-config', formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      });
      const newProjectId = response.data.data.project_id;
      setSaveStatus({ type: 'success', message: 'Config uploaded! Redirecting...' });
      setTimeout(() => {
        navigate(`/canvas/${newProjectId}`);
        window.location.reload();
      }, 1500);
    } catch (error) {
      console.error('Upload failed:', error);
      setSaveStatus({ type: 'error', message: 'Failed to upload YAML config.' });
    } finally {
      setIsSavingProject(false);
    }
  };

  return (
    <div className="canvas-container" ref={reactFlowWrapper} style={{ height: '100vh', width: '100vw' }}>
      <ReactFlowProvider>
        <Flow
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
              <button
                onClick={handleSaveProject}
                className="btn"
                disabled={isSavingProject}
                style={{
                  width: '100%',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  gap: '10px',
                  background: 'linear-gradient(135deg, #059669, #10b981)',
                  border: 'none',
                  padding: '12px',
                  fontWeight: 600,
                  boxShadow: '0 4px 12px rgba(16, 185, 129, 0.2)'
                }}
              >
                {isSavingProject ? <Loader2 className="animate-spin" size={18} /> : <Save size={18} />}
                {isSavingProject ? 'Saving...' : 'Save Project'}
              </button>


              <div style={{ height: '1px', background: 'var(--border-color)', margin: '4px 0' }} />
              <BlockPalette />
            </div>
          </Panel>

          <AnimatePresence>
            {saveStatus && (
              <motion.div
                initial={{ y: -50, opacity: 0, x: '-50%' }}
                animate={{ y: 0, opacity: 1, x: '-50%' }}
                exit={{ y: -50, opacity: 0, x: '-50%' }}
                style={{
                  position: 'fixed',
                  top: '20px',
                  left: '50%',
                  zIndex: 1000,
                  background: saveStatus.type === 'success' ? '#059669' : '#ef4444',
                  color: '#fff',
                  padding: '12px 24px',
                  borderRadius: '12px',
                  boxShadow: '0 10px 25px rgba(0,0,0,0.2)',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '10px',
                  fontWeight: 500
                }}
              >
                {saveStatus.type === 'success' ? <Save size={18} /> : <AlertCircle size={18} />}
                {saveStatus.message}
              </motion.div>
            )}
          </AnimatePresence>

          <AnimatePresence>
            {(selectedNode || selectedEdge) && (
              <Panel position="top-right" style={{ margin: '20px' }}>
                <motion.div
                  initial={{ x: 320, opacity: 0 }} animate={{ x: 0, opacity: 1 }} exit={{ x: 320, opacity: 0 }}
                  style={{
                    background: 'var(--panel-bg)', backdropFilter: 'blur(16px)', padding: '24px', borderRadius: '24px',
                    border: '1px solid var(--border-color)', boxShadow: '0 20px 40px rgba(0,0,0,0.4)', width: '300px', color: '#fff',
                    maxHeight: 'calc(100vh - 80px)', overflowY: 'auto'
                  }}
                >
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                      <Settings size={20} style={{ color: '#3b82f6' }} />
                      <h3 style={{ margin: 0, fontSize: '1.2rem' }}>{selectedNode ? 'Block Config' : 'Link Config'}</h3>
                    </div>
                    <button onClick={() => { setSelectedNode(null); setSelectedEdge(null); }} style={{ background: 'transparent', border: 'none', color: 'var(--text-secondary)', cursor: 'pointer' }}>
                      <X size={20} />
                    </button>
                  </div>

                  {isLoadingConfig ? (
                    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '12px', padding: '40px 0' }}>
                      <Loader2 className="animate-spin" size={32} style={{ color: '#3b82f6' }} />
                      <p style={{ color: 'var(--text-secondary)', fontSize: '0.9rem' }}>Loading Configuration...</p>
                    </div>
                  ) : configError ? (
                    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '12px', padding: '40px 0', color: '#ef4444', textAlign: 'center' }}>
                      <AlertCircle size={32} />
                      <p style={{ fontSize: '0.9rem' }}>{configError}</p>
                      <button
                        onClick={() => setSelectedNode({ ...selectedNode! })}
                        style={{ background: 'rgba(239, 68, 68, 0.1)', border: '1px solid #ef4444', color: '#ef4444', padding: '8px 16px', borderRadius: '8px', cursor: 'pointer' }}
                      >
                        Retry
                      </button>
                    </div>
                  ) : selectedNode ? (
                    <>
                      <div className="input-group" style={{ marginBottom: '16px' }}>
                        <label style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', textTransform: 'uppercase' }}>Type</label>
                        <div style={{ background: 'rgba(59, 130, 246, 0.1)', color: '#3b82f6', padding: '10px', borderRadius: '12px', textAlign: 'center', fontWeight: 600 }}>
                          {configSchema?.label || selectedNode.data.type?.replace('_', ' ')}
                        </div>
                      </div>

                      <div style={{ height: '1px', background: 'var(--border-color)', margin: '20px 0' }} />

                      {renderDynamicFields()}
                      {renderCustomFields()}

                      <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', marginTop: '24px' }}>
                        <input
                          type="file"
                          id="node-config-upload"
                          style={{ display: 'none' }}
                          accept=".yaml,.yml"
                          onChange={handleUploadNodeConfig}
                        />
                        <button
                          onClick={() => document.getElementById('node-config-upload')?.click()}
                          className="btn"
                          style={{ 
                            width: '100%', 
                            display: 'flex', 
                            alignItems: 'center', 
                            justifyContent: 'center',
                            gap: '8px',
                            background: 'rgba(59, 130, 246, 0.1)',
                            border: '1px solid rgba(59, 130, 246, 0.3)',
                            color: '#3b82f6'
                          }}
                        >
                          <Upload size={16} /> Upload YAML Config
                        </button>

                        <button
                          onClick={saveNodeConfig}
                          className="btn"
                          style={{ width: '100%', background: '#3b82f6', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px' }}
                          disabled={isSaving}
                        >
                          <Save size={16} /> {isSaving ? 'Saving...' : 'Save Configuration'}
                        </button>
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
                      <div className="input-group" style={{ marginBottom: '16px' }}>
                        <label style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', textTransform: 'uppercase' }}>To</label>
                        <div style={{ fontSize: '0.8rem', padding: '8px', background: 'rgba(0,0,0,0.2)', borderRadius: '8px', fontFamily: 'monospace' }}>{selectedEdge.target}</div>
                      </div>

                      <div style={{ height: '1px', background: 'var(--border-color)', margin: '20px 0' }} />

                      <div className="input-group" style={{ marginBottom: '16px' }}>
                        <label style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', textTransform: 'uppercase', display: 'flex', alignItems: 'center', gap: '6px' }}>
                          <Cpu size={14} /> Connection Hook (Go)
                        </label>
                        <textarea
                          value={selectedEdge.data?.hook_code || ''}
                          onChange={(e) => {
                            const newCode = e.target.value;
                            setEdges(eds => eds.map(edge => edge.id === selectedEdge.id ? { ...edge, data: { ...edge.data, hook_code: newCode } } : edge));
                            setSelectedEdge(prev => prev ? { ...prev, data: { ...prev.data, hook_code: newCode } } : null);
                          }}
                          placeholder="// Intercept request here\nfunc Handle(req *http.Request) {\n  // your logic\n}"
                          style={{
                            width: '100%', minHeight: '150px', background: 'rgba(0,0,0,0.3)', border: '1px solid var(--border-color)',
                            borderRadius: '12px', color: '#fff', padding: '12px', fontSize: '0.85rem', fontFamily: 'monospace',
                            marginTop: '8px', outline: 'none', resize: 'vertical'
                          }}
                        />
                      </div>

                      <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', marginTop: '12px' }}>
                        <button onClick={handleDeleteEdge} className="btn" style={{ width: '100%', background: '#ef4444', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px' }}>
                          <Trash2 size={16} /> Delete Connection
                        </button>
                      </div>
                    </>
                  )}
                </motion.div>
              </Panel>
            )}
          </AnimatePresence>

          <button 
            className="connection-btn" 
            onClick={handleGenerateHelm}
            disabled={isGeneratingHelm}
            style={{
            position: 'fixed', bottom: '40px', left: '50%', transform: 'translateX(-50%)',
            background: 'linear-gradient(135deg, #3b82f6, #8b5cf6)', color: '#fff', padding: '16px 32px', borderRadius: '100px',
            border: 'none', display: 'flex', alignItems: 'center', gap: '12px', fontSize: '1rem', fontWeight: 600,
            cursor: 'pointer', boxShadow: '0 10px 40px rgba(59, 130, 246, 0.4)', zIndex: 10
          }}>
            {isGeneratingHelm ? <Loader2 className="animate-spin" size={20} /> : <Rocket size={20} />}
            {isGeneratingHelm ? 'Generating Chart...' : 'Deploy Architecture'}
          </button>

          <AnimatePresence>
            {isGeneratingHelm && (
              <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                style={{
                  position: 'fixed', top: 0, left: 0, width: '100vw', height: '100vh',
                  background: 'rgba(2, 6, 23, 0.85)', backdropFilter: 'blur(12px)',
                  display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center',
                  zIndex: 10000, color: '#fff', gap: '24px'
                }}
              >
                <div style={{ position: 'relative', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                  <motion.div
                    animate={{ rotate: 360 }}
                    transition={{ duration: 2, repeat: Infinity, ease: "linear" }}
                    style={{
                      width: '120px', height: '120px', borderRadius: '50%',
                      border: '4px solid transparent', borderTopColor: '#3b82f6', borderBottomColor: '#8b5cf6'
                    }}
                  />
                  <Package size={48} style={{ position: 'absolute', color: '#3b82f6' }} />
                </div>
                <div style={{ textAlign: 'center' }}>
                  <h2 style={{ fontSize: '1.8rem', fontWeight: 700, marginBottom: '8px', background: 'linear-gradient(to right, #fff, #94a3b8)', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent' }}>
                    Generating Infrastructure Chart
                  </h2>
                  <p style={{ color: 'rgba(255,255,255,0.5)', fontSize: '1rem' }}>
                    Analyzing {nodes.length} nodes and connection hooks...
                  </p>
                </div>
              </motion.div>
            )}
          </AnimatePresence>
        </Flow>
      </ReactFlowProvider>
    </div>
  );
};

const Flow = ({ children, ...props }: any) => {
  return <ReactFlow {...props}>{children}</ReactFlow>;
};
