import React, { useRef, useEffect } from 'react';
import * as THREE from 'three';
import { useFrame, useLoader } from '@react-three/fiber';
import { GLTFLoader } from 'three/addons/loaders/GLTFLoader';

const Item = ({ url, position, scale, rotation }) => {
  const meshRef = useRef(null);
  const gltf = useLoader(GLTFLoader, url);

  useFrame(() => {
    meshRef.current.rotation.y += 0.01;
  });

  return (
    <mesh ref={meshRef} position={position} scale={scale} rotation={rotation}>
      <primitive object={gltf.scene} />
    </mesh>
  );
};

export default Item;