export class ZoomPan {
    constructor(container, workspace, svg) {
        this.container = container;
        this.workspace = workspace;
        this.svg = svg;

        this.translateX = 60;
        this.translateY = 40;
        this.scale = 0.9;
        this.isPanning = false;
        this.startX = 0;
        this.startY = 0;

        this.initEvents();
        this.updateTransform();
    }

    updateTransform() {
        this.workspace.setAttribute('transform', `translate(${this.translateX}, ${this.translateY}) scale(${this.scale})`);
    }

    zoom(factor, mouseX = null, mouseY = null) {
        const prevScale = this.scale;
        this.scale = Math.max(0.15, Math.min(3, this.scale + factor));

        if (mouseX !== null && mouseY !== null) {
            this.translateX = mouseX - (mouseX - this.translateX) * (this.scale / prevScale);
            this.translateY = mouseY - (mouseY - this.translateY) * (this.scale / prevScale);
        }
        this.updateTransform();
    }

    zoomIn() {
        this.zoom(0.15);
    }

    zoomOut() {
        this.zoom(-0.15);
    }

    reset() {
        this.translateX = 60;
        this.translateY = 40;
        this.scale = 0.9;
        this.updateTransform();
    }

    initEvents() {
        this.container.addEventListener('mousedown', (e) => {
            if (e.target.closest('.goal-card') || e.target.closest('#details-drawer') || e.target.closest('#controls-panel')) return;
            this.isPanning = true;
            this.startX = e.clientX - this.translateX;
            this.startY = e.clientY - this.translateY;
        });

        window.addEventListener('mousemove', (e) => {
            if (!this.isPanning) return;
            this.translateX = e.clientX - this.startX;
            this.translateY = e.clientY - this.startY;
            this.updateTransform();
        });

        window.addEventListener('mouseup', () => {
            this.isPanning = false;
        });

        this.container.addEventListener('wheel', (e) => {
            if (e.target.closest('#details-drawer')) return;
            e.preventDefault();
            const rect = this.svg.getBoundingClientRect();
            const mouseX = e.clientX - rect.left;
            const mouseY = e.clientY - rect.top;
            const zoomIntensity = 0.05;
            const factor = e.deltaY < 0 ? zoomIntensity : -zoomIntensity;
            this.zoom(factor, mouseX, mouseY);
        }, { passive: false });
    }
}
