import './index.css';
import { Suspense, useEffect, useRef, useState } from 'react'
import { Canvas } from '@react-three/fiber'
import { OrbitControls, useGLTF } from '@react-three/drei'
import Item from './Item';
import Button from 'react-bootstrap/Button';
import axios from 'axios';
import { toast, ToastContainer } from "react-toastify";
import "react-toastify/dist/ReactToastify.css";

const ASSET_MGR_ENDPOINT = "//127.0.0.1:9050/pay"
const ASSET_STORE_ENDPOINT_ASSET = "//127.0.0.1:7049/asset"
const ASSET_STORE_ENDPOINT_WS = "ws://127.0.0.1:7049/ws"
function App() {
  const [iframeContent, setIframeContent] = useState('');
  const [showIframe, setShowIframe] = useState(false);
  const [asset, setAsset] = useState('');

  useEffect(() => {

    axios
    .get(ASSET_STORE_ENDPOINT_ASSET)
    .then((res) => {
        setAsset(res.data.txId)
    })

    const socket = new WebSocket(ASSET_STORE_ENDPOINT_WS);

    socket.onopen = () => {
      console.log('WebSocket connection established.');
    };

    socket.onmessage = (event) => {
      toast.success(event.data);
      if(event.data === "Payment Received."){
        setShowIframe(false);
        axios
        .get(ASSET_STORE_ENDPOINT_ASSET)
        .then((res) => {
            setAsset(res.data.txId);
        })
      }
    };

    return () => {
      socket.close();
    };
  },[]);

  let handleButtonClick = async () => {
    try {
      const response = await axios.get(ASSET_MGR_ENDPOINT, {
        params: {
          merchant: "16Uiu2HAmSAnQRySqJdCEWrz5JCygK3CW1eqxUL8aR2gLaaGoGAC5",
          amount: 1000,
          sid: asset,
        }
      });
      setIframeContent(response.data);
      setShowIframe(true);
    } catch (error) {
      console.error('Error fetching data from API:', error);
    }
  };

  const models = [
    { url: '/milk/scene.gltf', position: [0, 0, 0], scale: [2.5, 2.5, 2.5], rotation: [0, Math.PI / 2, 0] },
  ];

  return (
    <div className="App">
      <ToastContainer
        autoClose={500}
      />

      <div className="wrapper">
        <div className="card">
          <div className="product-canvas">
            <Canvas>
              <Suspense fallback={null}>
                <ambientLight />
                <spotLight intensity={0.9}
                  angle={0.1}
                  penumbra={1}
                  position={[10, 15, 10]}
                  castShadow />
                <OrbitControls enablePan={true}
                  enableZoom={true}
                  enableRotate={true} />
                <Item key={models[0].url} {...models[0]} />
              </Suspense>
            </Canvas>
          </div>
          <div>{asset}</div>
          <div className='button'>
            <div>
              <Button onClick={handleButtonClick}>Buy</Button>
            </div>
          </div>
          {showIframe && (
            <div className="popup">
              <div className="popup_overlay" onClick={() => setShowIframe(false)}></div>
              <div className="popup_content">
                <iframe srcDoc={iframeContent} className="iframe" title="API Content" />
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default App;




