import { useEffect, useState } from 'react';
import { Alignment, Fit, Layout, RuntimeLoader, useRive } from '@rive-app/react-canvas';
import wasmUrl from '@rive-app/canvas/rive.wasm?url';
import type { LoginCatProps } from './LoginMascot';

RuntimeLoader.setWasmUrl(wasmUrl);
RuntimeLoader.setWasmFallbackUrl(null);
const machine = 'Login Machine';
const artboard = 'cat_artboard';

export default function RiveLoginCat({ coverEyes, result, lookAtForm = false, onReady }: LoginCatProps) {
  const [failed, setFailed] = useState(false);
  const { rive, canvas, RiveComponent } = useRive({
    src: `${import.meta.env.BASE_URL}mascots/login-cat.riv`,
    artboard, stateMachine: machine, autoplay: true,
    layout: new Layout({ fit: Fit.Contain, alignment: Alignment.Center }),
    shouldDisableRiveListeners: true,
    onLoadError: () => setFailed(true),
  });

  useEffect(() => {
    onReady(Boolean(rive) && !failed);
    return () => onReady(false);
  }, [rive, failed, onReady]);

  useEffect(() => {
    if (!rive || failed) return;
    // 每次结果从统一姿态开始；重置后重新取输入，避免持有被销毁的引用。
    rive.reset({ artboard, stateMachine: machine, autoplay: true });
    if (result !== 'idle') rive.stateMachineInputs(machine)?.find(input => input.name === (result === 'success' ? 'success' : 'fail'))?.fire();
    rive.play(machine);
  }, [rive, result, failed]);

  useEffect(() => {
    if (!rive || failed) return;
    const hands = rive.stateMachineInputs(machine)?.find(input => input.name === 'is_hands_up');
    if (hands) hands.value = result === 'idle' && coverEyes;
  }, [rive, coverEyes, result, failed]);

  useEffect(() => {
    if (!rive || !canvas || failed) return;
    const gaze = rive.stateMachineInputs(machine)?.find(input => input.name === 'look_direction');
    if (!gaze) return;
    gaze.value = lookAtForm ? 1 : 0;
    if (lookAtForm || coverEyes || result !== 'idle' || !window.matchMedia('(pointer: fine)').matches) return;
    let frame = 0;
    let pointerX = 0;
    const onPointerMove = (event: PointerEvent) => {
      if (event.pointerType === 'touch') return;
      pointerX = event.clientX;
      if (frame) return;
      frame = requestAnimationFrame(() => {
        frame = 0;
        const bounds = canvas.getBoundingClientRect();
        const offset = pointerX - bounds.left - bounds.width / 2;
        const previous = Number(gaze.value);
        // 进入转头与回正使用不同阈值，避免指针在边界附近时反复切换。
        const next = Math.abs(offset) > bounds.width * 0.35 ? Math.sign(offset) : Math.abs(offset) < bounds.width * 0.2 ? 0 : previous;
        if (next !== previous) gaze.value = next;
      });
    };
    const onPointerLeave = () => { cancelAnimationFrame(frame); frame = 0; gaze.value = 0; };
    window.addEventListener('pointermove', onPointerMove);
    document.documentElement.addEventListener('pointerleave', onPointerLeave);
    return () => {
      cancelAnimationFrame(frame);
      window.removeEventListener('pointermove', onPointerMove);
      document.documentElement.removeEventListener('pointerleave', onPointerLeave);
    };
  }, [rive, canvas, failed, lookAtForm, coverEyes, result]);

  return <>
    {(!rive || failed) && <img className="login-mascot-fallback" src={`${import.meta.env.BASE_URL}mascots/login-cat.png`} alt="" />}
    {!failed && <RiveComponent className={`login-mascot-canvas${rive ? ' is-ready' : ''}`} />}
  </>;
}
