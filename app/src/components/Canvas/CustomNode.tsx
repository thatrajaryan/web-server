import React from 'react';
import { Handle, Position } from 'reactflow';

export const CustomNode = ({ data, selected }: any) => {
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
