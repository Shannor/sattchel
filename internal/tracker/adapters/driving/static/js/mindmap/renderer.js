import { Layout } from './layout.js';

export class Renderer {
    constructor(mindmap, linksContainer, nodesContainer) {
        this.mindmap = mindmap;
        this.linksContainer = linksContainer;
        this.nodesContainer = nodesContainer;
    }

    clear() {
        this.linksContainer.innerHTML = '';
        this.nodesContainer.innerHTML = '';
    }

    render(goalsMap) {
        this.clear();

        Object.values(goalsMap).forEach(node => {
            const isRoot = !node.parent || !node.parent.targetId;

            // Draw circuit traces and solder pads to children
            node.children.forEach(child => {
                let startX, endX;
                if (child.side === 'left') {
                    startX = node.x;
                    endX = child.x + Layout.nodeWidth;
                } else {
                    startX = node.x + (isRoot ? Layout.cpuWidth : Layout.nodeWidth);
                    endX = child.x;
                }

                const startY = node.y + (isRoot ? Layout.cpuHeight : Layout.nodeHeight) / 2;
                const endY = child.y + Layout.nodeHeight / 2;
                const x_mid = (startX + endX) / 2;

                // Orthogonal routing: Horizontal -> Vertical -> Horizontal
                const pathStr = `M ${startX} ${startY} H ${x_mid} V ${endY} H ${endX}`;
                
                const path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
                path.setAttribute('d', pathStr);
                path.setAttribute('class', `connection status-${child.status || 'draft'}`);
                this.linksContainer.appendChild(path);

                // Place small circular solder pads at connection points and bends
                const pads = [];
                pads.push({ x: startX, y: startY });
                pads.push({ x: endX, y: endY });
                if (startY !== endY) {
                    pads.push({ x: x_mid, y: startY });
                    pads.push({ x: x_mid, y: endY });
                }

                pads.forEach(p => {
                    const circle = document.createElementNS('http://www.w3.org/2000/svg', 'circle');
                    circle.setAttribute('cx', p.x);
                    circle.setAttribute('cy', p.y);
                    circle.setAttribute('r', '4');
                    circle.setAttribute('class', `solder-pad status-${child.status || 'draft'}`);
                    this.linksContainer.appendChild(circle);
                });
            });

            // Create HTML foreignObject node card
            const fo = document.createElementNS('http://www.w3.org/2000/svg', 'foreignObject');
            
            const width = isRoot ? Layout.cpuWidth : Layout.nodeWidth;
            const height = isRoot ? Layout.cpuHeight : Layout.nodeHeight;

            fo.setAttribute('x', node.x);
            fo.setAttribute('y', node.y);
            fo.setAttribute('width', width);
            fo.setAttribute('height', height);
            fo.setAttribute('class', isRoot ? 'root-card-wrapper' : 'goal-card-wrapper');

            const statusClass = node.status || 'draft';
            const impactStr = node.impact || 'unknown';
            const effortStr = node.effort || 'unknown';
            const statusLabel = statusClass.replace('-', ' ');

            if (isRoot) {
                fo.innerHTML = `
                    <div xmlns="http://www.w3.org/1999/xhtml" class="root-card">
                        <div class="root-card-title" title="${node.name}">${node.name}</div>
                    </div>
                `;
            } else {
                fo.innerHTML = `
                    <div xmlns="http://www.w3.org/1999/xhtml" class="goal-card status-${statusClass}" draggable="true">
                        <div class="goal-title" title="${node.name}">${node.name}</div>
                        <div class="goal-meta">
                            <span class="badge status-${statusClass}">${statusLabel}</span>
                            ${impactStr !== 'unknown' ? `<span class="badge impact-badge">${impactStr}</span>` : ''}
                            ${effortStr !== 'unknown' ? `<span class="badge">${effortStr}</span>` : ''}
                        </div>
                    </div>
                `;
            }

            const card = fo.querySelector(isRoot ? '.root-card' : '.goal-card');

            // Click listener for details drawer
            card.addEventListener('click', () => {
                this.mindmap.drawer.show(node);
            });

            // Drag and drop event listeners for non-root goal cards
            if (!isRoot) {
                card.addEventListener('dragstart', (e) => this.mindmap.dragDrop.handleDragStart(e, node.id));
                card.addEventListener('dragover', (e) => this.mindmap.dragDrop.handleDragOver(e));
                card.addEventListener('dragleave', (e) => this.mindmap.dragDrop.handleDragLeave(e));
                card.addEventListener('dragend', (e) => this.mindmap.dragDrop.handleDragEnd(e));
                card.addEventListener('drop', (e) => this.mindmap.dragDrop.handleDrop(e, node.id));
            }

            this.nodesContainer.appendChild(fo);
        });
    }
}
